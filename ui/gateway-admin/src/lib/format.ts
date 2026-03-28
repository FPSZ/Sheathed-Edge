export function formatTime(value?: unknown): string {
  if (!value) return "-";
  if (typeof value === "number") {
    return new Date(value).toLocaleString();
  }
  if (typeof value === "string") {
    const parsed = Date.parse(value);
    if (!Number.isNaN(parsed)) {
      return new Date(parsed).toLocaleString();
    }
    return value;
  }
  return String(value);
}

export function formatJson(value: unknown): string {
  return JSON.stringify(value, null, 2);
}
