import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Pass, PassEvent } from "@/lib/types";

import { PassRow } from "./PassRow";

type Props = {
  passes: Pass[];
  loading: boolean;
  eventsByPass: Record<number, PassEvent[]>;
  expandedPassID: number | null;
  eventsLoadingID: number | null;
  onCopyShare: (pass: Pass) => void;
  onDuplicate: (pass: Pass) => void;
  onToggleEvents: (passID: number) => void;
  onRevoke: (passID: number) => void;
};

// PassList is the right-rail collection: a small KPI strip plus the
// scrolling list of rows. The page hands it state and callbacks.
export function PassList({
  passes,
  loading,
  eventsByPass,
  expandedPassID,
  eventsLoadingID,
  onCopyShare,
  onDuplicate,
  onToggleEvents,
  onRevoke,
}: Props) {
  const activeCount = passes.filter((p) => p.status === "active").length;
  const revokedCount = passes.filter((p) => p.revoked_at || p.status === "revoked").length;
  const totalOpens = passes.reduce((sum, p) => sum + p.open_count, 0);

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle>Recent passes</CardTitle>
        <Badge variant="secondary">{passes.length} total</Badge>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-3 gap-2">
          <Metric label="Active" value={String(activeCount)} />
          <Metric label="Revoked" value={String(revokedCount)} />
          <Metric label="Opened" value={String(totalOpens)} />
        </div>

        {loading && (
          <div className="space-y-2">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-20 w-full" />
          </div>
        )}
        {!loading && passes.length === 0 && (
          <div className="rounded-md border border-dashed border-border p-4 text-sm">
            <p className="font-medium text-foreground">No passes yet.</p>
            <p className="text-muted-foreground">Create a pass to start sharing.</p>
          </div>
        )}
        <div className="space-y-2">
          {passes.map((pass) => (
            <PassRow
              key={pass.id}
              pass={pass}
              events={eventsByPass[pass.id]}
              eventsLoading={eventsLoadingID === pass.id}
              expanded={expandedPassID === pass.id}
              onCopyShare={() => onCopyShare(pass)}
              onDuplicate={() => onDuplicate(pass)}
              onToggleEvents={() => onToggleEvents(pass.id)}
              onRevoke={() => onRevoke(pass.id)}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-surface px-3 py-2">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="text-lg font-semibold">{value}</p>
    </div>
  );
}
