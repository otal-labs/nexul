import { boardGridTemplate } from "@/components/board/boardGrid";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";

interface BoardSkeletonProps {
  columns: BoardStatus[];
}

// The board's own shape while it loads: a toolbar line, the stage row and one lane of card ghosts, so the page never jumps from a spinner into a grid.
export const BoardSkeleton = ({ columns }: BoardSkeletonProps) => (
  <div aria-hidden className="mt-3 min-h-0 flex-1 animate-pulse">
    <div className="flex items-center gap-2">
      <div className="h-9 w-60 rounded-md bg-muted/40" />
      <div className="h-9 w-20 rounded-md bg-muted/40" />
      <div className="ml-auto h-9 w-24 rounded-md bg-muted/40" />
    </div>
    <div className="mt-3 grid gap-2" style={{ gridTemplateColumns: boardGridTemplate(columns, true) }}>
      {columns.map((column) => (
        <div key={column.id} className="flex h-8 items-center gap-1.5 px-2">
          <span className="size-3.5 rounded-full bg-muted/50" />
          <span className="h-2.5 w-20 rounded-full bg-muted/50" />
        </div>
      ))}
    </div>
    <div className="mt-2 grid gap-2" style={{ gridTemplateColumns: boardGridTemplate(columns, true) }}>
      {columns.map((column) => (
        <div key={column.id} className={cn("flex flex-col gap-1.5 rounded-lg p-2", column.kind === "done" ? "bg-transparent" : "bg-well-shade/50")}>
          {[0, 1, 2].map((ghost) => (
            <div key={ghost} className={cn("h-[76px] rounded-lg bg-card", ghost > 0 && "opacity-60")} />
          ))}
        </div>
      ))}
    </div>
  </div>
);
