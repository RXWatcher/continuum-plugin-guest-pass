export function formatUsageStat(label: string, current: number, limit: number): string {
  return `${label} ${current}/${limit > 0 ? String(limit) : "∞"}`;
}

export function buildPassRowMeta(pass: {
  target_type: string;
  target_id: string;
  effective_expires_at: string;
  max_resolution: string;
}): string[] {
  return [formatPassTargetContext(pass.target_type, pass.target_id), `Expires ${formatShortDate(pass.effective_expires_at)}`, pass.max_resolution];
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

export function formatPassTargetContext(targetType: string, targetID: string): string {
  const label = humanizeTargetType(targetType);
  return `${label} #${targetID}`;
}

export function guestPassStatusMessage(status: string): string | null {
  switch (status) {
    case "expired":
      return "This guest pass has expired.";
    case "revoked":
      return "This guest pass has been revoked.";
    case "device_limit_reached":
      return "This guest pass has reached its device limit.";
    case "open_limit_reached":
      return "This guest pass has reached its open limit.";
    case "play_limit_reached":
      return "This guest pass has reached its play limit.";
    case "concurrent_stream_limit_reached":
      return "Too many viewers are using this guest pass right now. Try again shortly.";
    case "ip_locked":
      return "This guest pass is locked to the network where it was first opened.";
    case "ip_not_allowed":
      return "This guest pass is not available from your network.";
    case "country_not_allowed":
      return "This guest pass is not available in your region.";
    case "pin_required":
      return "Enter the access PIN to continue.";
    case "not_found":
      return "This guest pass could not be found.";
    default:
      return null;
  }
}

export function formatShortDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

function humanizeTargetType(targetType: string): string {
  if (targetType === "media_file") return "File";
  const normalized = targetType.replace(/[_-]+/g, " ").trim().toLowerCase();
  if (!normalized) return "Item";
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}
