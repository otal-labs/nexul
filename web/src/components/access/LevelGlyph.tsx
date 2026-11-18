import { cn } from "@/lib/utils";

interface LevelGlyphProps {
  level: number;
  rungs?: number;
  className?: string;
}

// Read, Write and Delete from the left: filled up to the level, muted above it, outlined where the area has no such rung.
export const LevelGlyph = ({ level, rungs = 3, className }: LevelGlyphProps) => (
  <span aria-hidden className={cn("inline-flex shrink-0 gap-px", className)}>
    {Array.from({ length: 3 }, (_, slot) => {
      const missing = slot >= rungs;
      return (
        <span
          key={slot}
          className={cn(
            "h-2 w-1.5",
            slot === 0 && "rounded-l-[2px]",
            slot === 2 && "rounded-r-[2px]",
            missing && "border border-muted-foreground/45",
            !missing && (slot < level ? "bg-foreground/75" : "bg-muted-foreground/60"),
          )}
        />
      );
    })}
  </span>
);
