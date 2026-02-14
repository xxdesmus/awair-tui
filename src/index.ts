#!/usr/bin/env node

/**
 * Awair TUI — real-time air quality monitoring for Awair sensors.
 *
 * Usage:
 *   awair-tui                        # auto-discover devices via mDNS
 *   awair-tui 192.168.1.10           # connect to specific IP(s)
 *   awair-tui 192.168.1.10 10.0.0.5  # multiple devices
 *   awair-tui --no-discovery 10.0.0.5
 *   awair-tui --interval 5           # poll every 5 seconds (default: 10)
 */

import { fetchAirData, fetchDeviceConfig, type AwairDevice } from "./api.js";
import { discoverAwairDevices } from "./discovery.js";
import { Dashboard } from "./ui.js";

// --- CLI argument parsing ---

interface Options {
  ips: string[];
  discovery: boolean;
  interval: number;
}

function parseArgs(argv: string[]): Options {
  const opts: Options = {
    ips: [],
    discovery: true,
    interval: 10,
  };

  const args = argv.slice(2);
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === "--no-discovery") {
      opts.discovery = false;
    } else if (arg === "--interval" || arg === "-i") {
      const next = args[++i];
      const n = Number(next);
      if (isNaN(n) || n < 1) {
        console.error(`Invalid interval: ${next}`);
        process.exit(1);
      }
      opts.interval = n;
    } else if (arg === "--help" || arg === "-h") {
      console.log(`
Awair TUI — Real-time air quality monitoring

Usage:
  awair-tui [options] [ip ...]

Options:
  --no-discovery    Disable mDNS auto-discovery
  --interval, -i N  Polling interval in seconds (default: 10)
  --help, -h        Show this help message

Examples:
  awair-tui                            Auto-discover devices
  awair-tui 192.168.1.100              Connect to specific device
  awair-tui -i 5 192.168.1.100 .200    Poll every 5s, two devices
`);
      process.exit(0);
    } else if (arg.startsWith("-")) {
      console.error(`Unknown option: ${arg}`);
      process.exit(1);
    } else {
      opts.ips.push(arg);
    }
  }

  return opts;
}

// --- Device management ---

const devices: Map<string, AwairDevice> = new Map();

function addDevice(ip: string, name?: string): AwairDevice {
  const existing = devices.get(ip);
  if (existing) {
    if (name && existing.name !== name) {
      existing.name = name;
    }
    return existing;
  }

  const device: AwairDevice = {
    ip,
    name: name || ip,
    data: null,
    config: null,
    lastError: null,
    lastUpdate: null,
  };
  devices.set(ip, device);
  return device;
}

async function pollDevice(device: AwairDevice): Promise<void> {
  try {
    device.data = await fetchAirData(device.ip);
    device.lastError = null;
    device.lastUpdate = new Date();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    device.lastError = message;
  }
}

async function fetchConfig(device: AwairDevice): Promise<void> {
  try {
    device.config = await fetchDeviceConfig(device.ip);
    if (device.config.device_uuid && device.name === device.ip) {
      device.name = device.config.device_uuid;
    }
  } catch {
    // Config endpoint may not be available on all devices
  }
}

async function pollAllDevices(): Promise<void> {
  const promises = Array.from(devices.values()).map((d) => pollDevice(d));
  await Promise.allSettled(promises);
}

// --- Main ---

async function main(): Promise<void> {
  const opts = parseArgs(process.argv);
  const dashboard = new Dashboard();

  // Add manually specified devices
  for (const ip of opts.ips) {
    const device = addDevice(ip);
    dashboard.log(`Added device: ${ip}`);
    fetchConfig(device);
  }

  // Start mDNS discovery
  let discoveryHandle: { stop: () => void } | null = null;

  function startDiscovery(): void {
    if (!opts.discovery) return;
    if (discoveryHandle) discoveryHandle.stop();

    dashboard.log("Starting mDNS discovery...");

    discoveryHandle = discoverAwairDevices(
      (found) => {
        if (!devices.has(found.ip)) {
          dashboard.log(
            `{green-fg}Discovered:{/green-fg} ${found.name} at ${found.ip}`
          );
          const device = addDevice(found.ip, found.name);
          fetchConfig(device);
          // Immediately poll the new device
          pollDevice(device).then(() => {
            dashboard.updateDevices(Array.from(devices.values()));
          });
        }
      },
      (err) => {
        dashboard.log(`{yellow-fg}Discovery warning:{/yellow-fg} ${err.message}`);
      }
    );
  }

  startDiscovery();

  // Key bindings
  dashboard.onKey("r", async () => {
    dashboard.log("Refreshing...");
    await pollAllDevices();
    dashboard.updateDevices(Array.from(devices.values()));
  });

  dashboard.onKey("a", () => {
    dashboard.showPrompt("Enter device IP address:", (value) => {
      if (value) {
        // Basic IP format validation
        if (/^[\d.]+$/.test(value) || value.includes(":")) {
          const device = addDevice(value);
          dashboard.log(`Added device: ${value}`);
          fetchConfig(device);
          pollDevice(device).then(() => {
            dashboard.updateDevices(Array.from(devices.values()));
          });
        } else {
          dashboard.log(`{red-fg}Invalid IP: ${value}{/red-fg}`);
        }
      }
    });
  });

  dashboard.onKey("d", () => {
    startDiscovery();
  });

  // Initial render
  dashboard.updateDevices(Array.from(devices.values()));

  // Initial poll if we have devices
  if (devices.size > 0) {
    await pollAllDevices();
    dashboard.updateDevices(Array.from(devices.values()));
  }

  // Periodic polling
  setInterval(async () => {
    if (devices.size > 0) {
      await pollAllDevices();
      dashboard.updateDevices(Array.from(devices.values()));
    }
  }, opts.interval * 1000);

  // Handle clean shutdown
  process.on("SIGINT", () => {
    if (discoveryHandle) discoveryHandle.stop();
    dashboard.destroy();
    process.exit(0);
  });

  process.on("SIGTERM", () => {
    if (discoveryHandle) discoveryHandle.stop();
    dashboard.destroy();
    process.exit(0);
  });
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
