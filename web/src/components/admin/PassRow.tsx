import { Activity, Copy, MoreHorizontal, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { absoluteURL } from "@/lib/api";
import { buildPassRowMeta, buildPassRowTags, formatUsageStat } from "@/lib/presentation";
import type { Pass, PassEvent } from "@/lib/types";

import { EventTimeline } from "./EventTimeline";

type Props = {
  pass: Pass;
  events: PassEvent[] | undefined;
  eventsLoading: boolean;
  expanded: boolean;
  onCopyShare: () => void;
  onDuplicate: () => void;
  onToggleEvents: () => void;
  onRevoke: () => void;
};

// PassRow renders a single guest pass: title, metadata strip, usage
// counters, restriction badges, and the row-action menu. Activity
// timeline expands inline.
export function PassRow({
  pass,
  events,
  eventsLoading,
  expanded,
  onCopyShare,
  onDuplicate,
  onToggleEvents,
  onRevoke,
}: Props) {
  const meta = buildPassRowMeta(pass);
  const tags = buildPassRowTags(pass);
  const statusTone = pass.status === "active" ? "default" : pass.status === "revoked" ? "destructive" : "secondary";

  return (
    <article className="space-y-2 rounded-md border border-border bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="truncate font-medium">{pass.title}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground">{meta.join(" · ")}</p>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm" variant="ghost" className="h-8 w-8 p-0">
              <MoreHorizontal className="h-4 w-4" />
              <span className="sr-only">Open row menu</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {pass.share_url && (
              <DropdownMenuItem onSelect={onCopyShare}>
                <Copy className="mr-2 h-4 w-4" />
                Copy share link
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onSelect={onDuplicate}>
              <Copy className="mr-2 h-4 w-4" />
              Duplicate
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={onToggleEvents}>
              <Activity className="mr-2 h-4 w-4" />
              {expanded ? "Hide activity" : "Show activity"}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onRevoke} className="text-destructive focus:text-destructive">
              <Trash2 className="mr-2 h-4 w-4" />
              Revoke
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <div className="flex flex-wrap items-center gap-1.5 text-xs">
        <Badge variant={statusTone}>{pass.status}</Badge>
        <Badge variant="outline">{formatUsageStat("Opens", pass.open_count, pass.max_opens)}</Badge>
        <Badge variant="outline">{formatUsageStat("Plays", pass.play_count, pass.max_plays)}</Badge>
        {tags.map((tag) => (
          <Badge key={tag} variant="secondary">
            {tag}
          </Badge>
        ))}
      </div>

      {pass.share_url && (
        <p className="truncate font-mono text-xs text-muted-foreground" title={absoluteURL(pass.share_url)}>
          {absoluteURL(pass.share_url)}
        </p>
      )}

      {expanded && <EventTimeline events={events} loading={eventsLoading} />}
    </article>
  );
}
