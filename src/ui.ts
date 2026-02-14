/**
 * Terminal UI for displaying Awair sensor data.
 * Uses blessed for the TUI framework.
 */

import blessed from "blessed";
import {
  type AwairDevice,
  type SensorKey,
  OPTIMAL_RANGES,
  rateSensorValue,
} from "./api.js";

const COLORS: Record<string, string> = {
  good: "green",
  fair: "yellow",
  poor: "red",
};

const SCORE_LABELS: [number, string, string][] = [
  [80, "Good", "green"],
  [60, "Fair", "yellow"],
  [0, "Poor", "red"],
];

function scoreColor(score: number): string {
  for (const [threshold, , color] of SCORE_LABELS) {
    if (score >= threshold) return color;
  }
  return "red";
}

function scoreLabel(score: number): string {
  for (const [threshold, label] of SCORE_LABELS) {
    if (score >= threshold) return label;
  }
  return "Poor";
}

function formatValue(key: SensorKey, value: number): string {
  const range = OPTIMAL_RANGES[key];
  if (key === "temp") return `${value.toFixed(1)}${range.unit}`;
  if (key === "humid") return `${value.toFixed(1)}${range.unit}`;
  return `${Math.round(value)} ${range.unit}`;
}

function sensorBar(key: SensorKey, value: number, width: number): string {
  const range = OPTIMAL_RANGES[key];
  let ratio: number;

  if (key === "temp") {
    // Map 10-40°C range to bar
    ratio = Math.max(0, Math.min(1, (value - 10) / 30));
  } else if (key === "humid") {
    ratio = Math.max(0, Math.min(1, value / 100));
  } else if (key === "co2") {
    ratio = Math.max(0, Math.min(1, value / 2500));
  } else if (key === "voc") {
    ratio = Math.max(0, Math.min(1, value / 1500));
  } else {
    // pm25
    ratio = Math.max(0, Math.min(1, value / 100));
  }

  const filled = Math.round(ratio * width);
  const rating = rateSensorValue(key, value);
  const color = COLORS[rating];

  const bar = "█".repeat(filled) + "░".repeat(width - filled);
  return `{${color}-fg}${bar}{/${color}-fg}`;
}

function scoreGauge(score: number, width: number): string {
  const ratio = Math.max(0, Math.min(1, score / 100));
  const filled = Math.round(ratio * width);
  const color = scoreColor(score);
  return `{${color}-fg}${"█".repeat(filled)}${"░".repeat(width - filled)}{/${color}-fg}`;
}

export class Dashboard {
  private screen: blessed.Widgets.Screen;
  private headerBox: blessed.Widgets.BoxElement;
  private deviceBoxes: Map<string, blessed.Widgets.BoxElement> = new Map();
  private logBox: blessed.Widgets.BoxElement;
  private statusBar: blessed.Widgets.BoxElement;
  private logMessages: string[] = [];

  constructor() {
    this.screen = blessed.screen({
      smartCSR: true,
      title: "Awair TUI",
      fullUnicode: true,
    });

    // Header
    this.headerBox = blessed.box({
      parent: this.screen,
      top: 0,
      left: 0,
      width: "100%",
      height: 3,
      tags: true,
      content: this.renderHeader(),
      style: {
        fg: "white",
        bg: "black",
      },
    });

    // Log box (bottom area)
    this.logBox = blessed.box({
      parent: this.screen,
      bottom: 1,
      left: 0,
      width: "100%",
      height: 6,
      label: " Log ",
      tags: true,
      border: { type: "line" },
      scrollable: true,
      alwaysScroll: true,
      style: {
        fg: "white",
        border: { fg: "gray" },
        label: { fg: "gray" },
      },
    });

    // Status bar
    this.statusBar = blessed.box({
      parent: this.screen,
      bottom: 0,
      left: 0,
      width: "100%",
      height: 1,
      tags: true,
      content:
        " {bold}q{/bold} Quit  {bold}r{/bold} Refresh  {bold}a{/bold} Add device  {bold}d{/bold} Discovery",
      style: {
        fg: "white",
        bg: "#333333",
      },
    });

    // Key bindings
    this.screen.key(["escape", "q", "C-c"], () => {
      process.exit(0);
    });
  }

  private renderHeader(): string {
    const title = "{bold}{cyan-fg} ☁  Awair TUI {/cyan-fg}{/bold}";
    const subtitle = "{gray-fg}Real-time air quality monitoring{/gray-fg}";
    return `${title}  ${subtitle}`;
  }

  onKey(
    keys: string | string[],
    handler: (ch: string, key: blessed.Widgets.Events.IKeyEventArg) => void
  ): void {
    this.screen.key(keys, handler);
  }

  log(message: string): void {
    const ts = new Date().toLocaleTimeString();
    this.logMessages.push(`{gray-fg}${ts}{/gray-fg} ${message}`);
    if (this.logMessages.length > 100) {
      this.logMessages.shift();
    }
    this.logBox.setContent(this.logMessages.slice(-4).join("\n"));
    this.render();
  }

  showPrompt(
    label: string,
    callback: (value: string | null) => void
  ): void {
    const prompt = blessed.textbox({
      parent: this.screen,
      top: "center",
      left: "center",
      width: 50,
      height: 3,
      label: ` ${label} `,
      tags: true,
      border: { type: "line" },
      style: {
        fg: "white",
        bg: "black",
        border: { fg: "cyan" },
        label: { fg: "cyan" },
      },
      inputOnFocus: true,
    });

    prompt.focus();
    prompt.readInput((err, value) => {
      prompt.destroy();
      this.render();
      if (err || value === undefined) {
        callback(null);
      } else {
        callback(value.trim() || null);
      }
    });
    this.render();
  }

  updateDevices(devices: AwairDevice[]): void {
    // Remove old device boxes
    for (const [ip, box] of this.deviceBoxes) {
      if (!devices.find((d) => d.ip === ip)) {
        box.destroy();
        this.deviceBoxes.delete(ip);
      }
    }

    const contentTop = 3;
    const contentBottom = 7; // log box height + status bar
    const availableHeight =
      (this.screen.height as number) - contentTop - contentBottom;
    const availableWidth = this.screen.width as number;

    if (devices.length === 0) {
      // Show a "no devices" message
      if (!this.deviceBoxes.has("__empty__")) {
        const emptyBox = blessed.box({
          parent: this.screen,
          top: contentTop,
          left: 0,
          width: "100%",
          height: availableHeight,
          tags: true,
          content: this.renderEmptyState(),
          valign: "middle",
          align: "center",
          style: { fg: "gray" },
        });
        this.deviceBoxes.set("__empty__", emptyBox);
      }
      this.render();
      return;
    }

    // Remove empty state if it exists
    const emptyBox = this.deviceBoxes.get("__empty__");
    if (emptyBox) {
      emptyBox.destroy();
      this.deviceBoxes.delete("__empty__");
    }

    // Calculate grid layout
    const cols = devices.length <= 2 ? devices.length : Math.min(3, devices.length);
    const rows = Math.ceil(devices.length / cols);
    const boxWidth = Math.floor(availableWidth / cols);
    const boxHeight = Math.max(14, Math.floor(availableHeight / rows));

    devices.forEach((device, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);

      let box = this.deviceBoxes.get(device.ip);
      if (!box) {
        box = blessed.box({
          parent: this.screen,
          tags: true,
          border: { type: "line" },
          style: {
            fg: "white",
            border: { fg: "cyan" },
            label: { fg: "cyan", bold: true },
          },
        });
        this.deviceBoxes.set(device.ip, box);
      }

      box.top = contentTop + row * boxHeight;
      box.left = col * boxWidth;
      box.width = col === cols - 1 ? availableWidth - col * boxWidth : boxWidth;
      box.height = boxHeight;
      box.setLabel(` ${device.name} (${device.ip}) `);
      box.setContent(this.renderDeviceContent(device, boxWidth - 4));
    });

    this.render();
  }

  private renderEmptyState(): string {
    return [
      "",
      "{bold}No Awair devices found{/bold}",
      "",
      "Searching via mDNS discovery...",
      "",
      'Press {bold}a{/bold} to manually add a device IP',
      'Press {bold}d{/bold} to restart discovery',
      "Press {bold}q{/bold} to quit",
    ].join("\n");
  }

  private renderDeviceContent(device: AwairDevice, width: number): string {
    if (device.lastError && !device.data) {
      return `\n  {red-fg}Error: ${device.lastError}{/red-fg}\n\n  Retrying...`;
    }

    if (!device.data) {
      return "\n  {yellow-fg}Connecting...{/yellow-fg}";
    }

    const d = device.data;
    const barWidth = Math.max(10, width - 28);
    const lines: string[] = [];

    // Awair Score
    const sc = scoreColor(d.score);
    const sl = scoreLabel(d.score);
    lines.push(
      `  {bold}Awair Score{/bold}    {${sc}-fg}{bold}${d.score}{/bold} ${sl}{/${sc}-fg}`
    );
    lines.push(`  ${scoreGauge(d.score, barWidth + 16)}`);
    lines.push("");

    // Sensor readings
    const sensors: [SensorKey, number][] = [
      ["temp", d.temp],
      ["humid", d.humid],
      ["co2", d.co2],
      ["voc", d.voc],
      ["pm25", d.pm25],
    ];

    for (const [key, value] of sensors) {
      const range = OPTIMAL_RANGES[key];
      const rating = rateSensorValue(key, value);
      const color = COLORS[rating];
      const valStr = formatValue(key, value);
      const label = range.label.padEnd(13);
      const valPad = valStr.padStart(12);
      const bar = sensorBar(key, value, barWidth);

      lines.push(`  {bold}${label}{/bold} {${color}-fg}${valPad}{/${color}-fg}  ${bar}`);
    }

    // Timestamp
    if (device.lastUpdate) {
      lines.push("");
      lines.push(
        `  {gray-fg}Updated: ${device.lastUpdate.toLocaleTimeString()}{/gray-fg}`
      );
    }

    return lines.join("\n");
  }

  render(): void {
    this.screen.render();
  }

  destroy(): void {
    this.screen.destroy();
  }
}
