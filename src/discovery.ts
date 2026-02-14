/**
 * mDNS device discovery for Awair sensors.
 * Browses for _http._tcp services with hostnames matching "awair-*".
 */

import { Bonjour, type Service } from "bonjour-service";

export interface DiscoveredDevice {
  name: string;
  ip: string;
  port: number;
  host: string;
}

export function discoverAwairDevices(
  onDevice: (device: DiscoveredDevice) => void,
  onError?: (err: Error) => void
): { stop: () => void } {
  const bonjour = new Bonjour();

  const browser = bonjour.find({ type: "http" }, (service: Service) => {
    // Awair devices advertise as _http._tcp with hostnames like awair-elem-XXXXXX
    const name = (service.name || "").toLowerCase();
    if (!name.startsWith("awair")) return;

    // Get the first IPv4 address
    const addresses = service.addresses || [];
    const ipv4 = addresses.find(
      (a: string) => a.includes(".") && !a.includes(":")
    );
    if (!ipv4) return;

    onDevice({
      name: service.name || name,
      ip: ipv4,
      port: service.port || 80,
      host: service.host || "",
    });
  });

  browser.on("error", (err: Error) => {
    if (onError) onError(err);
  });

  return {
    stop() {
      browser.stop();
      bonjour.destroy();
    },
  };
}
