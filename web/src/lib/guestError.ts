import type { APIError } from "@/lib/types";
import { guestPassStatusMessage } from "@/lib/presentation";

// guestFacingErrorMessage converts an API error into copy a recipient can
// understand. Prefers our static status copy over backend message text so
// users don't see internal codes.
export function guestFacingErrorMessage(err: unknown, fallback: string): string {
  const status =
    typeof err === "object" && err && "responseGuestStatus" in err
      ? (err as APIError).responseGuestStatus
      : undefined;
  const guestMessage = typeof status === "string" ? guestPassStatusMessage(status) : null;
  if (guestMessage) return guestMessage;
  return err instanceof Error ? err.message : fallback;
}
