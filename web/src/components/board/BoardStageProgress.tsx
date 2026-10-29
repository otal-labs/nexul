import { cn } from "@/lib/utils";
import { StatusKind, statusStage, type BoardStatus } from "@/models/Status";

interface BoardStageProgressProps {
  columns: BoardStatus[];
  counts: Record<string, number>;
  className?: string;
}

// The board's shape at a glance: one segment per stage, as wide as its share of the tickets on show.
export const BoardStageProgress = ({ columns, counts, className }: BoardStageProgressProps) => {
  const total = columns.reduce((sum, column) => sum + (counts[column.id] ?? 0), 0);
  if (columns.length === 0 || total === 0) return null;

  return (
    // Decorative: the stage row under it names every stage and counts it in words.
    <div aria-hidden className={cn("flex h-2 w-60 shrink-0 gap-0.5 max-lg:w-40 max-md:hidden", className)}>
      {columns.map((column) => {
        const count = counts[column.id] ?? 0;
        return (
          <span
            key={column.id}
            data-stage-segment={column.id}
            className={cn(
              // A stage with nothing on show keeps a sliver, so the bar still maps to the row beside it.
              "h-full min-w-1 rounded-full",
              statusStage(column.kind).dot,
              count === 0 ? "opacity-25" : column.kind === StatusKind.Backlog && "opacity-50",
            )}
            style={{ flexGrow: count, flexBasis: 0 }}
          />
        );
      })}
    </div>
  );
};
