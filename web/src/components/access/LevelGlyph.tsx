import { cn } from "@/lib/utils";

interface LevelGlyphProps {
  level: number;
  rungs?: number;
  className?: string;
}

// Read, Write and Delete from the left: filled up to the level, an outline above it, and a baseline tick where the area has no such rung.
export const LevelGlyph = ({ level, rungs = 3, className }: LevelGlyphProps) => (
  <span aria-hidden className={cn("inline-flex h-2 shrink-0 items-end gap-px", className)}>
    {Array.from({ length: 3 }, (_, slot) => {
      const missing = slot >= rungs;
      return (
        <span
          key={slot}
          className={cn(
            "w-1.5",
            !missing && "h-2",
            slot === 0 && "rounded-l-[2px]",
            slot === 2 && "rounded-r-[2px]",
            !missing && slot < level && "bg-foreground/75",
            !missing && slot >= level && "border border-muted-foreground",
            missing && "h-px bg-muted-foreground/60",
          )}
        />
      );
    })}
  </span>
);
