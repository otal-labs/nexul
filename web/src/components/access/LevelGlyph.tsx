import { cn } from "@/lib/utils";

interface LevelGlyphProps {
  level: number;
  rungs?: number;
  // "end" puts a shorter ladder's empty slots before it, so the glyph ends where the word beside it starts.
  align?: "start" | "end";
  className?: string;
}

// The level strip in miniature: every rung up to the level fills; a shorter ladder keeps the three-rung width so a column of them lines up.
export const LevelGlyph = ({ level, rungs = 3, align = "start", className }: LevelGlyphProps) => {
  const pad = align === "end" ? 3 - rungs : 0;
  return (
    <span aria-hidden className={cn("inline-flex shrink-0 gap-px", className)}>
      {Array.from({ length: 3 }, (_, slot) => {
        const rung = slot - pad;
        return (
          <span
            key={slot}
            className={cn(
              "h-2 w-1.5",
              rung === 0 && "rounded-l-[2px]",
              rung === rungs - 1 && "rounded-r-[2px]",
              rung < level ? "bg-foreground/75" : "bg-muted-foreground/60",
              (rung < 0 || rung >= rungs) && "invisible",
            )}
          />
        );
      })}
    </span>
  );
};
