import { api } from "@/lib/api";
import type { Pass, PlayResponse } from "@/lib/types";

// previewPass reads the pass without recording an "opened" event. The
// admin uses this when deciding whether to prompt for a PIN.
export async function previewPass(token: string): Promise<Pass> {
  const data = await api<{ pass: Pass }>(`/api/public/passes/${token}`);
  return data.pass;
}

export type OpenPassRequest = {
  pin?: string;
  device_id: string;
};

export async function openPass(token: string, body: OpenPassRequest): Promise<Pass> {
  const data = await api<{ pass: Pass }>(`/api/public/passes/${token}/open`, {
    method: "POST",
    body: JSON.stringify(body),
  });
  return data.pass;
}

export async function playPass(token: string, body: OpenPassRequest): Promise<PlayResponse> {
  return api<PlayResponse>(`/api/public/passes/${token}/play`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}
