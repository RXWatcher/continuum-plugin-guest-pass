import { api } from "@/lib/api";
import type { CreateResponse, Pass, PassEvent } from "@/lib/types";

export type CreatePassRequest = {
  title: string;
  target_type: "media_file";
  target_id: string;
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

export async function listPasses(): Promise<Pass[]> {
  const data = await api<{ passes: Pass[] }>("/api/admin/passes");
  return data.passes ?? [];
}

export async function createPass(req: CreatePassRequest): Promise<CreateResponse> {
  return api<CreateResponse>("/api/admin/passes", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function revokePass(id: number): Promise<void> {
  await api(`/api/admin/passes/${id}/revoke`, { method: "POST" });
}

export async function listPassEvents(id: number): Promise<PassEvent[]> {
  const data = await api<{ events: PassEvent[] }>(`/api/admin/passes/${id}/events`);
  return data.events ?? [];
}
