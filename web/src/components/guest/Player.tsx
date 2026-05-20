import { visibleWatermark } from "@/lib/share";
import type { PlayResponse } from "@/lib/types";

type Props = {
  playback: PlayResponse;
};

// Player frames the HTML5 video element with the optional watermark
// overlay (logo and/or text). The actual stream URL is minted by the
// host and is single-use.
export function Player({ playback }: Props) {
  if (!playback.stream_url) return null;
  const showOverlay = visibleWatermark(playback.pass.watermark_mode);
  const streamSrc = new URL(playback.stream_url, window.location.origin).toString();
  return (
    <div className="relative overflow-hidden rounded-lg border border-border bg-black">
      <video
        controls
        autoPlay
        src={streamSrc}
        className="block aspect-video w-full"
      />
      {showOverlay && (
        <>
          {playback.logo_url && (
            <img
              src={playback.logo_url}
              alt=""
              className="pointer-events-none absolute right-3 top-3 max-h-16 opacity-70 mix-blend-screen"
            />
          )}
          {playback.watermark && (
            <div className="pointer-events-none absolute bottom-3 left-3 rounded bg-black/40 px-2 py-1 text-xs text-white/90">
              {playback.watermark}
            </div>
          )}
        </>
      )}
    </div>
  );
}
