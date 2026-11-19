import { cn } from "@/lib/utils";

interface LevelGlyphProps {
  level: number;
  rungs?: number;
  className?: string;
}

const SLOTS = ["Read", "Write", "Delete"];

// What each slot means, for the glyph's tooltip: "Read and Write held, Delete not offered here".
const glyphTitle = (level: number, rungs: number): string => {
  const held = SLOTS.slice(0, level);
  const open = SLOTS.slice(level, rungs);
  const absent = SLOTS.slice(rungs);
  return [held.length > 0 && `${held.join(" and ")} held`, open.length > 0 && `${open.join(" and ")} not held`, absent.length > 0 && `${absent.join(" and ")} not offered here`]
    .filter(Boolean)
    .join(", ");
};

// Read, Write and Delete from the left: filled up to the level, an outline above it, and a baseline tick where the area has no such rung.
export const LevelGlyph = ({ level, rungs = 3, className }: LevelGlyphProps) => (
  <span aria-hidden title={glyphTitle(level, rungs)} className={cn("inline-flex h-2 shrink-0 items-end gap-px", className)}>
    {Array.from({ length: 3 }, (_, slot) => {
      const missing = slot >= rungs;
      return (
        <span
          key={slot}
          className={cn(
            "w-1.5",
            !missing && "h-2",
            slot === 0 && slot < level && "rounded-l-[2px]",
            slot === 2 && slot < level && "rounded-r-[2px]",
            !missing && slot < level && "bg-foreground/75",
            !missing && slot >= level && "border border-muted-foreground",
            missing && "h-px bg-muted-foreground",
          )}
        />
      );
    })}
  </span>
);
