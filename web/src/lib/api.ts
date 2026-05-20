import { mountPath } from "@/lib/mountPath";
import { authHeaders } from "@/lib/authToken";
import type { APIError } from "@/lib/types";

// api<T> is the single fetch helper used by every typed API module.
// Resolves with parsed JSON on 2xx; throws an APIError shaped with the
// HTTP status and the backend's error envelope on non-2xx so callers can
// branch on `error.responseGuestStatus` for the public guest-side flow.
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${mountPath()}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(),
      ...(init?.headers ?? {}),
    },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const message = data?.error?.message || data?.message || `Request failed (${res.status})`;
    const error = new Error(message) as APIError;
    error.responseStatus = res.status;
    error.responseCode = data?.error?.code;
    error.responseGuestStatus = data?.status;
    throw error;
  }
  return data as T;
}

export function absoluteURL(url: string): string {
  if (url.startsWith("/") && mountPath()) {
    return new URL(`${mountPath()}${url}`, window.location.origin).toString();
  }
  return new URL(url, window.location.href).toString();
}
