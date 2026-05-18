export type PassDraft = {
  title: string;
  target_id?: string;
  note: string;
  expires_in_hours: number;
  valid_hours_after_first_open: number;
  max_opens: number;
  max_plays: number;
  max_watch_minutes: number;
  max_concurrent_streams: number;
  max_devices: number;
  max_resolution: string;
  allow_downloads: boolean;
  allow_direct_play: boolean;
  lock_to_first_ip: boolean;
  require_pin: boolean;
  pin: string;
  disable_seeking: boolean;
  watermark_mode: string;
  watermark_profile: string;
  watermark_logo_url: string;
  ip_allowlist: string;
  country_allowlist: string;
  session_grace_minutes: number;
  per_item_play_count: boolean;
  geofence: string;
};

export type PassLike = Omit<PassDraft, "ip_allowlist" | "country_allowlist" | "geofence" | "note"> & {
  note?: string;
  ip_allowlist?: string | string[];
  country_allowlist?: string | string[];
  geofence?: string | string[];
  revoked_at?: string;
};

export type PassTemplate = {
  id: string;
  label: string;
  description: string;
  values: Partial<PassDraft>;
};

export const passTemplates: PassTemplate[] = [
  {
    id: "preview-24h",
    label: "24h preview",
    description: "Short, low-friction preview with one playback session.",
    values: {
      expires_in_hours: 24,
      valid_hours_after_first_open: 0,
      max_opens: 3,
      max_plays: 1,
      max_watch_minutes: 180,
      max_concurrent_streams: 1,
      max_devices: 1,
      require_pin: false,
      disable_seeking: false,
      watermark_mode: "none",
    },
  },
  {
    id: "one-time-screening",
    label: "One-time screening",
    description: "PIN-gated pass for a single viewer and one playback.",
    values: {
      expires_in_hours: 72,
      valid_hours_after_first_open: 8,
      max_opens: 2,
      max_plays: 1,
      max_watch_minutes: 240,
      max_concurrent_streams: 1,
      max_devices: 1,
      require_pin: true,
      lock_to_first_ip: true,
      disable_seeking: true,
      watermark_mode: "visible",
      watermark_profile: "Guest pass {{pass_id}} · {{ip}} · {{time}}",
    },
  },
  {
    id: "press-screener",
    label: "Press screener",
    description: "Watermarked review access with a longer calendar window.",
    values: {
      expires_in_hours: 168,
      valid_hours_after_first_open: 0,
      max_opens: 6,
      max_plays: 2,
      max_watch_minutes: 300,
      max_concurrent_streams: 1,
      max_devices: 2,
      require_pin: true,
      disable_seeking: false,
      watermark_mode: "visible",
      watermark_profile: "Screener {{pass_id}} · {{ip}} · {{time}}",
    },
  },
  {
    id: "family-share",
    label: "Family share",
    description: "A small household-friendly share with moderate limits.",
    values: {
      expires_in_hours: 168,
      valid_hours_after_first_open: 48,
      max_opens: 12,
      max_plays: 4,
      max_watch_minutes: 600,
      max_concurrent_streams: 2,
      max_devices: 4,
      require_pin: false,
      lock_to_first_ip: false,
      disable_seeking: false,
      watermark_mode: "none",
    },
  },
];

export function buildPolicySummary(form: PassDraft): string[] {
  const items = [
    `Expires in ${form.expires_in_hours || 24} ${plural(form.expires_in_hours || 24, "hour")}`,
    form.valid_hours_after_first_open > 0
      ? `Valid for ${form.valid_hours_after_first_open} ${plural(form.valid_hours_after_first_open, "hour")} after first open`
      : "Valid until calendar expiry",
    limitPhrase(form.max_plays, "play"),
    limitPhrase(form.max_watch_minutes, "watch minute"),
    limitPhrase(form.max_concurrent_streams, "concurrent stream"),
    limitPhrase(form.max_devices, "device"),
  ];
  if (form.require_pin) items.push("PIN required");
  if (form.lock_to_first_ip) items.push("Locks to first IP");
  if (form.disable_seeking) items.push("Seeking disabled");
  if (form.allow_downloads) items.push("Downloads allowed");
  if (form.allow_direct_play) items.push("Direct play allowed");
  if (form.watermark_mode && form.watermark_mode !== "none") {
    items.push(`${labelWatermark(form.watermark_mode)} watermark`);
  }
  if (listHasItems(form.ip_allowlist)) items.push(`IP allowlist: ${compactList(form.ip_allowlist)}`);
  if (listHasItems(form.country_allowlist)) items.push(`Allowed countries: ${compactList(form.country_allowlist)}`);
  if (listHasItems(form.geofence)) items.push(`Geofence: ${compactList(form.geofence)}`);
  return items;
}

export function duplicatePassForm(pass: PassLike): PassDraft {
  return {
    title: `Copy of ${pass.title}`,
    target_id: pass.target_id,
    note: pass.note || "",
    expires_in_hours: 24,
    valid_hours_after_first_open: pass.valid_hours_after_first_open,
    max_opens: pass.max_opens,
    max_plays: pass.max_plays,
    max_watch_minutes: pass.max_watch_minutes,
    max_concurrent_streams: pass.max_concurrent_streams,
    max_devices: pass.max_devices,
    max_resolution: pass.max_resolution,
    allow_downloads: pass.allow_downloads,
    allow_direct_play: pass.allow_direct_play,
    lock_to_first_ip: pass.lock_to_first_ip,
    require_pin: pass.require_pin,
    pin: "",
    disable_seeking: pass.disable_seeking,
    watermark_mode: pass.watermark_mode || "none",
    watermark_profile: pass.watermark_profile || "",
    watermark_logo_url: pass.watermark_logo_url || "",
    ip_allowlist: listString(pass.ip_allowlist),
    country_allowlist: listString(pass.country_allowlist),
    session_grace_minutes: pass.session_grace_minutes,
    per_item_play_count: pass.per_item_play_count,
    geofence: listString(pass.geofence),
  };
}

export function eventTone(eventType: string): "success" | "danger" | "neutral" {
  if (eventType.includes("rejected") || eventType.includes("failed")) return "danger";
  if (["opened", "grant_minted", "play_attempted"].includes(eventType)) return "success";
  return "neutral";
}

function limitPhrase(value: number, label: string): string {
  if (value <= 0) return `Unlimited ${label}s`;
  return `${value} ${plural(value, label)}`;
}

function plural(value: number, label: string): string {
  return value === 1 ? label : `${label}s`;
}

function labelWatermark(mode: string): string {
  return mode.replace(/[_-]+/g, " ").replace(/^\w/, (char) => char.toUpperCase());
}

function compactList(value: string | string[]): string {
  const items = Array.isArray(value) ? value : value.split(/[\n,]+/);
  return items.map((item) => item.trim()).filter(Boolean).join(", ");
}

function listHasItems(value: string | string[]): boolean {
  return compactList(value) !== "";
}

function listString(value: unknown): string {
  return Array.isArray(value) ? value.join(", ") : "";
}
