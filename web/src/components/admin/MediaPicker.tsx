import { useEffect, useState } from "react";
import { Search } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { searchCatalog } from "@/api/catalog";
import type { MediaItem } from "@/lib/types";

import { MediaThumb } from "./MediaThumb";

type MediaType = "all" | "movie" | "episode";

type Props = {
  selected: MediaItem | null;
  onSelect: (item: MediaItem) => void;
};

// MediaPicker debounces input by 250ms and aborts in-flight requests on
// every keystroke so old responses can't clobber the current query.
export function MediaPicker({ selected, onSelect }: Props) {
  const [query, setQuery] = useState("");
  const [type, setType] = useState<MediaType>("all");
  const [results, setResults] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const trimmed = query.trim();
    if (trimmed.length < 2) {
      setResults([]);
      setLoading(false);
      setError("");
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setLoading(true);
      setError("");
      searchCatalog({ query: trimmed, type, signal: controller.signal })
        .then((data) => setResults(data.items ?? []))
        .catch((err) => {
          if (err instanceof DOMException && err.name === "AbortError") return;
          setError(err instanceof Error ? err.message : "Media search failed");
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false);
        });
    }, 250);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [query, type]);

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-[1fr_140px] gap-3">
        <div className="space-y-1.5">
          <Label htmlFor="media-search">Search media</Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" size={16} />
            <Input
              id="media-search"
              className="pl-8"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Title, episode, collection..."
            />
          </div>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="media-type">Type</Label>
          <Select value={type} onValueChange={(v) => setType(v as MediaType)}>
            <SelectTrigger id="media-type">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="movie">Movies</SelectItem>
              <SelectItem value="episode">Episodes</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {selected && (
        <div className="flex items-center gap-3 rounded-md border border-border bg-card p-3">
          <MediaThumb title={selected.title} posterURL={selected.poster_url} />
          <div className="min-w-0">
            <p className="truncate font-medium">{selected.title}</p>
            <p className="text-xs text-muted-foreground">{mediaMeta(selected)}</p>
          </div>
        </div>
      )}

      {error && <p className="text-sm text-destructive">{error}</p>}
      {loading && <Skeleton className="h-16 w-full" />}
      {!loading && query.trim().length >= 2 && results.length === 0 && !error && (
        <div className="rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground">
          <p className="font-medium text-foreground">No matches found.</p>
          <p>Try a broader title or switch media type.</p>
        </div>
      )}

      {results.length > 0 && (
        <ul className="max-h-72 space-y-1 overflow-auto rounded-md border border-border bg-card p-1">
          {results.map((item) => {
            const isSelected = selected?.content_id === item.content_id;
            return (
              <li key={`${item.type}:${item.content_id}`}>
                <button
                  type="button"
                  onClick={() => onSelect(item)}
                  disabled={!item.playable}
                  className={`flex w-full items-center gap-3 rounded-sm px-2 py-2 text-left transition-colors hover:bg-surface-hover disabled:opacity-50 disabled:cursor-not-allowed ${
                    isSelected ? "bg-surface-hover" : ""
                  }`}
                >
                  <MediaThumb title={item.title} posterURL={item.poster_url} className="h-12 w-9" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{item.title}</span>
                    <span className="block truncate text-xs text-muted-foreground">{mediaMeta(item)}</span>
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function mediaMeta(item: MediaItem): string {
  const parts: string[] = [item.type];
  if (item.year) parts.push(String(item.year));
  if (item.runtime_minutes) parts.push(`${item.runtime_minutes} min`);
  if (item.content_rating) parts.push(item.content_rating);
  if (item.content_id) parts.push(item.content_id);
  if (item.media_file_id) parts.push(`file ${item.media_file_id}`);
  return parts.join(" · ");
}
