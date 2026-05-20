import { Skeleton } from "@/components/ui/skeleton";
import { eventTone } from "@/lib/passTools";
import { formatShortDate } from "@/lib/presentation";
import type { PassEvent } from "@/lib/types";

type Props = {
  events: PassEvent[] | undefined;
  loading: boolean;
};

const toneClasses: Record<ReturnType<typeof eventTone>, string> = {
  success: "border-l-success/50 bg-success/5",
  danger: "border-l-destructive/50 bg-destructive/5",
  neutral: "border-l-border bg-surface",
};

// EventTimeline renders the audit history below a pass row when the
// operator clicks "Activity". Shows up to 200 events from the backend.
export function EventTimeline({ events, loading }: Props) {
  if (loading) {
    return (
      <div className="mt-2 space-y-1">
        <Skeleton className="h-6 w-full" />
        <Skeleton className="h-6 w-3/4" />
      </div>
    );
  }
  if (!events || events.length === 0) {
    return (
      <div className="mt-2 rounded-md border border-dashed border-border p-3 text-sm">
        <p className="font-medium text-foreground">No activity recorded.</p>
        <p className="text-muted-foreground">Open or play this pass to start an audit trail.</p>
      </div>
    );
  }
  return (
    <ol className="mt-2 space-y-1">
      {events.map((event) => (
        <li
          key={event.id}
          className={`rounded-sm border-l-2 px-3 py-1.5 text-xs ${toneClasses[eventTone(event.type)]}`}
        >
          <span className="font-medium">{event.type.replace(/_/g, " ")}</span>
          <span className="ml-2 text-muted-foreground">
            {formatShortDate(event.created_at)}
            {event.ip ? ` · ${event.ip}` : ""}
          </span>
        </li>
      ))}
    </ol>
  );
}
