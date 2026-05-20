import { buildGuestFacts } from "@/lib/presentation";
import type { Pass } from "@/lib/types";

type Props = {
  pass: Pass;
  compact?: boolean;
};

// GuestFacts is the entitlement table shown above the play button:
// resolution, devices allowed, watch time cap, plays remaining.
// Compact variant is rendered alongside playback in the player frame.
export function GuestFacts({ pass, compact }: Props) {
  const facts = buildGuestFacts(pass);
  const grid = compact ? "grid-cols-4" : "grid-cols-2 sm:grid-cols-4";
  return (
    <dl className={`grid gap-3 text-sm ${grid}`}>
      {facts.map(([label, value]) => (
        <div key={label} className="rounded-md border border-border bg-card px-3 py-2">
          <dt className="text-xs uppercase tracking-wide text-muted-foreground">{label}</dt>
          <dd className="mt-0.5 font-medium">{value}</dd>
        </div>
      ))}
    </dl>
  );
}
