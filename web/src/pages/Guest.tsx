import { useEffect, useMemo, useState } from "react";
import { ExternalLink } from "lucide-react";
import { useParams } from "react-router";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { GuestFacts } from "@/components/guest/GuestFacts";
import { GuestHero } from "@/components/guest/GuestHero";
import { PinForm } from "@/components/guest/PinForm";
import { Player } from "@/components/guest/Player";

import { openPass, playPass, previewPass } from "@/api/public";
import { guestDeviceID } from "@/lib/deviceId";
import { guestFacingErrorMessage } from "@/lib/guestError";
import type { Pass, PlayResponse } from "@/lib/types";

type Phase = "loading" | "needs_pin" | "active" | "playing" | "unavailable";

// Guest is the recipient-facing page. State machine:
//
//   loading  →  preview pass
//             ├─ requires_pin → needs_pin → (unlock) → active
//             └─ otherwise    → open      →           active
//   active   →  (play)        →           playing  (video element shown)
//   any failure                →          unavailable
export function Guest() {
  const { token: tokenParam } = useParams();
  const token = tokenParam ?? "";
  const deviceId = useMemo(() => guestDeviceID(), []);

  const [phase, setPhase] = useState<Phase>("loading");
  const [pass, setPass] = useState<Pass | null>(null);
  const [pin, setPin] = useState("");
  const [message, setMessage] = useState("");
  const [playback, setPlayback] = useState<PlayResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function start() {
      try {
        const preview = await previewPass(token);
        if (cancelled) return;
        if (preview.require_pin) {
          // The server withholds the full pass (target, policy, watermark)
          // until the PIN is verified, so we only prompt here. The complete
          // pass arrives from openPass once the guest unlocks.
          setPhase("needs_pin");
          return;
        }
        const opened = await openPass(token, { device_id: deviceId });
        if (cancelled) return;
        setPass(opened);
        setPhase("active");
      } catch (err) {
        if (cancelled) return;
        setMessage(guestFacingErrorMessage(err, "This guest pass is unavailable."));
        setPhase("unavailable");
      }
    }
    void start();
    return () => {
      cancelled = true;
    };
  }, [token, deviceId]);

  async function unlock() {
    try {
      const opened = await openPass(token, { pin, device_id: deviceId });
      setPass(opened);
      setPhase("active");
      setMessage("");
    } catch (err) {
      setMessage(guestFacingErrorMessage(err, "Enter the access PIN to continue."));
    }
  }

  async function play() {
    try {
      const data = await playPass(token, { pin, device_id: deviceId });
      setPass(data.pass);
      setMessage("");
      if (data.stream_url) {
        setPlayback(data);
        setPhase("playing");
        return;
      }
      setMessage(data.message || "Playback is not available.");
    } catch (err) {
      setMessage(guestFacingErrorMessage(err, "Playback is not available."));
    }
  }

  return (
    <main className="min-h-[100dvh] bg-background text-foreground">
      <div className="mx-auto max-w-3xl space-y-6 px-4 py-10">
        <GuestHero pass={pass} />

        {phase === "loading" && (
          <div role="status" aria-live="polite">
            <Skeleton className="h-32 w-full" />
          </div>
        )}

        {phase === "needs_pin" && (
          <PinForm pin={pin} onChange={setPin} onSubmit={unlock} />
        )}

        {phase === "active" && pass && (
          <div className="space-y-4">
            <GuestFacts pass={pass} />
            <Button
              onClick={play}
              disabled={pass.status !== "active"}
              className="w-full"
              size="lg"
            >
              <ExternalLink className="mr-2 h-4 w-4" />
              Start playback
            </Button>
          </div>
        )}

        {phase === "playing" && playback && pass && (
          <div className="space-y-4">
            <Player playback={playback} />
            <GuestFacts pass={pass} compact />
          </div>
        )}

        {phase === "unavailable" && (
          <div className="rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground">
            {message || "This guest pass is unavailable."}
          </div>
        )}

        {pass && message && phase !== "unavailable" && (
          <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
            {message}
          </div>
        )}
      </div>
    </main>
  );
}
