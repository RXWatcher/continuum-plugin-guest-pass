export function formatUsageStat(label: string, current: number, limit: number): string {
  return `${label} ${current}/${limit > 0 ? String(limit) : "∞"}`;
}

export function buildPassRowMeta(pass: {
  effective_expires_at: string;
  max_resolution: string;
}): string[] {
  return [`Expires ${formatShortDate(pass.effective_expires_at)}`, pass.max_resolution];
}

export function buildPassRowTags(pass: {
  require_pin: boolean;
  lock_to_first_ip: boolean;
  allow_downloads: boolean;
  allow_direct_play: boolean;
}): string[] {
  const tags: string[] = [];
  if (pass.require_pin) tags.push("PIN");
  if (pass.lock_to_first_ip) tags.push("IP lock");
  if (pass.allow_downloads) tags.push("Downloads");
  if (pass.allow_direct_play) tags.push("Direct play");
  return tags;
}

export function buildGuestFacts(pass: {
  max_resolution: string;
  max_devices: number;
  max_watch_minutes: number;
  max_plays: number;
}): Array<[string, string]> {
  return [
    ["Resolution", pass.max_resolution],
    ["Devices", pass.max_devices > 0 ? String(pass.max_devices) : "∞"],
    ["Watch time", pass.max_watch_minutes > 0 ? `${pass.max_watch_minutes} min` : "∞ min"],
    ["Play limit", pass.max_plays > 0 ? String(pass.max_plays) : "∞"],
  ];
}

export function formatShortDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}
