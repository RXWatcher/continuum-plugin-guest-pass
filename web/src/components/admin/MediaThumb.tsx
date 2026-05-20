type Props = {
  title: string;
  posterURL?: string;
  className?: string;
};

// MediaThumb shows the poster if we have one, otherwise a single-letter
// fallback tile. Used everywhere a media item is rendered (picker,
// selected card, guest hero).
export function MediaThumb({ title, posterURL, className }: Props) {
  const initial = (title || "?").slice(0, 1).toUpperCase();
  const base = "flex items-center justify-center rounded-md bg-muted text-muted-foreground text-lg font-semibold overflow-hidden shrink-0";
  if (posterURL) {
    return <img src={posterURL} alt="" className={`${base} object-cover ${className ?? "h-16 w-12"}`} />;
  }
  return <span className={`${base} ${className ?? "h-16 w-12"}`}>{initial}</span>;
}
