const units = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

/**
 * Formats a byte count into a human-readable string using base-1024 units.
 * Mirrors Go `pkg/core.FormatBytes`.
 */
export function formatBytes(n: number, precision = 1): string {
  if (n === 0) return "0 B";

  const k = 1024;
  const i = Math.min(Math.floor(Math.log(n) / Math.log(k)), units.length - 1);

  const size = n / Math.pow(k, i);
  return `${size.toFixed(Math.max(precision, 0))} ${units[i]}`;
}
