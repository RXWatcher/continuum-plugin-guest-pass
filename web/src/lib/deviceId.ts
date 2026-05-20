// Stable per-browser device id used by the open/play endpoints to enforce
// MaxDevices. Persisted to localStorage so the same browser session keeps
// the same id across reloads; falls back to a one-shot UUID for tabs
// where localStorage is unavailable (private browsing).
const KEY = "continuum.guestPass.deviceId";

export function guestDeviceID(): string {
  try {
    const existing = window.localStorage.getItem(KEY);
    if (existing) return existing;
    const id = crypto.randomUUID();
    window.localStorage.setItem(KEY, id);
    return id;
  } catch {
    return crypto.randomUUID();
  }
}
