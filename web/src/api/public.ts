import { api } from "@/lib/api";
import type { Pass, PlayResponse } from "@/lib/types";

// PassPreview is the response of the public preview endpoint. For a
// PIN-protected pass the server withholds the full pass payload behind the
// PIN gate and returns only enough to render the PIN prompt
// ({ require_pin, status, title }). For an open pass it returns the full
// decorated pass.
export type PassPreview =
  | { require_pin: true; status: string; title: string; pass?: undefined }
  | { require_pin?: false; pass: Pass };

// previewPass reads the pass without recording an "opened" event. The
// guest page uses this to decide whether to prompt for a PIN first.
export async function previewPass(token: string): Promise<PassPreview> {
  return api<PassPreview>(`/api/public/passes/${token}`);
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
