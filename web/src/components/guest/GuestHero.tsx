import { formatShortDate } from "@/lib/presentation";
import type { Pass } from "@/lib/types";

import { MediaThumb } from "@/components/admin/MediaThumb";

type Props = {
  pass: Pass | null;
};

// GuestHero is the recipient-facing top card: poster + title + note +
// human-readable expiry. Stays mounted across loading/PIN/play states so
// the page feels continuous.
export function GuestHero({ pass }: Props) {
  const title = pass?.title ?? "Guest Pass";
  return (
    <div className="flex items-start gap-4">
      <MediaThumb title={title} className="h-24 w-16" />
      <div className="min-w-0 space-y-1">
        <p className="text-xs uppercase tracking-wide text-muted-foreground">
          Silo guest pass
        </p>
        <h1 className="text-2xl font-semibold">{title}</h1>
        {pass?.note && <p className="text-sm text-muted-foreground">{pass.note}</p>}
        {pass && (
          <p className="text-xs text-muted-foreground">
            Available until {formatShortDate(pass.effective_expires_at)}
          </p>
        )}
      </div>
    </div>
  );
}
