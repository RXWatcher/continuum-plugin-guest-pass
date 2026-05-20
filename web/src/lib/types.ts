// Wire shapes shared between API client and pages. Field names match the
// JSON the Go backend emits (snake_case).

export type Pass = {
  id: number;
  title: string;
  target_type: string;
  target_id: string;
  note?: string;
  created_by: string;
  expires_at: string;
  effective_expires_at: string;
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
  disable_seeking: boolean;
  watermark_mode: string;
  watermark_profile: string;
  watermark_logo_url: string;
  ip_allowlist: string[];
  country_allowlist: string[];
  session_grace_minutes: number;
  per_item_play_count: boolean;
  geofence: string[];
  open_count: number;
  play_count: number;
  first_opened_at?: string;
  revoked_at?: string;
  created_at: string;
  status: string;
  share_url?: string;
};

export type CreateResponse = {
  pass: Pass;
  token: string;
  share_url: string;
};

export type PassEvent = {
  id: number;
  pass_id: number;
  type: string;
  ip?: string;
  user_agent?: string;
  attrs?: Record<string, unknown>;
  created_at: string;
};

export type PlayResponse = {
  message?: string;
  pass: Pass;
  stream_url?: string;
  play_method?: string;
  expires_at?: string;
  watermark?: string;
  logo_url?: string;
};

export type AppConfig = {
  public_base_url: string;
  audit_retention_days: number;
};

export type MediaItem = {
  content_id: string;
  media_file_id: number;
  type: string;
  title: string;
  year?: number;
  overview?: string;
  poster_url?: string;
  genres?: string[];
  runtime_minutes?: number;
  content_rating?: string;
  playable: boolean;
  external_id?: string;
  external_provider?: string;
};

export type CatalogSearchResponse = {
  items: MediaItem[];
  total: number;
};

export type APIError = Error & {
  responseStatus?: number;
  responseCode?: string;
  responseGuestStatus?: string;
};
