import { cn } from "@/lib/utils";

interface LevelGlyphProps {
  level: number;
  rungs?: number;
  className?: string;
}

// The level strip in miniature: every rung up to the level fills; a shorter ladder keeps the three-rung width so a column of them lines up.
export const LevelGlyph = ({ level, rungs = 3, className }: LevelGlyphProps) => (
  <span aria-hidden className={cn("inline-flex shrink-0 gap-px", className)}>
    {Array.from({ length: 3 }, (_, rung) => (
      <span
        key={rung}
        className={cn(
          "h-2 w-1.5 first:rounded-l-[2px]",
          rung === rungs - 1 && "rounded-r-[2px]",
          rung < level ? "bg-foreground/75" : "bg-foreground/12",
          rung >= rungs && "invisible",
        )}
      />
    ))}
  </span>
);
