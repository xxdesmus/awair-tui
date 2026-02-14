/**
 * Awair Local API client.
 * Polls /air-data/latest on each device for real-time sensor readings.
 */

export interface AwairSensorData {
  timestamp: string;
  score: number;
  dew_point?: number;
  temp: number;
  humid: number;
  abs_humid?: number;
  co2: number;
  co2_est?: number;
  co2_est_baseline?: number;
  voc: number;
  voc_baseline?: number;
  voc_h2_raw?: number;
  voc_ethanol_raw?: number;
  pm25: number;
  pm10_est?: number;
}

export interface AwairDeviceConfig {
  device_uuid?: string;
  wifi_mac?: string;
  ssid?: string;
  ip?: string;
  netmask?: string;
  gateway?: string;
  fw_version?: string;
  timezone?: string;
  display?: string;
  led?: object;
}

export interface AwairDevice {
  ip: string;
  name: string;
  data: AwairSensorData | null;
  config: AwairDeviceConfig | null;
  lastError: string | null;
  lastUpdate: Date | null;
}

export async function fetchAirData(ip: string): Promise<AwairSensorData> {
  const url = `http://${ip}/air-data/latest`;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);

  try {
    const res = await fetch(url, {
      headers: { Accept: "application/json" },
      signal: controller.signal,
    });
    if (!res.ok) {
      throw new Error(`HTTP ${res.status} ${res.statusText}`);
    }
    return (await res.json()) as AwairSensorData;
  } finally {
    clearTimeout(timeout);
  }
}

export async function fetchDeviceConfig(
  ip: string
): Promise<AwairDeviceConfig> {
  const url = `http://${ip}/settings/config/data`;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);

  try {
    const res = await fetch(url, {
      headers: { Accept: "application/json" },
      signal: controller.signal,
    });
    if (!res.ok) {
      throw new Error(`HTTP ${res.status} ${res.statusText}`);
    }
    return (await res.json()) as AwairDeviceConfig;
  } finally {
    clearTimeout(timeout);
  }
}

/** Optimal ranges per Awair's scoring methodology */
export const OPTIMAL_RANGES = {
  temp: { min: 20, max: 25, unit: "°C", label: "Temperature" },
  humid: { min: 40, max: 50, unit: "%", label: "Humidity" },
  co2: { min: 0, max: 600, unit: "ppm", label: "CO₂" },
  voc: { min: 0, max: 300, unit: "ppb", label: "VOC" },
  pm25: { min: 0, max: 12, unit: "µg/m³", label: "PM2.5" },
} as const;

export type SensorKey = keyof typeof OPTIMAL_RANGES;

/**
 * Returns a rating for a sensor value: "good", "fair", or "poor".
 */
export function rateSensorValue(
  key: SensorKey,
  value: number
): "good" | "fair" | "poor" {
  const range = OPTIMAL_RANGES[key];

  if (key === "temp") {
    if (value >= range.min && value <= range.max) return "good";
    const dist = value < range.min ? range.min - value : value - range.max;
    return dist <= 3 ? "fair" : "poor";
  }

  if (key === "humid") {
    if (value >= range.min && value <= range.max) return "good";
    const dist = value < range.min ? range.min - value : value - range.max;
    return dist <= 10 ? "fair" : "poor";
  }

  // For co2, voc, pm25 — lower is better
  if (value <= range.max) return "good";
  if (value <= range.max * 2) return "fair";
  return "poor";
}
