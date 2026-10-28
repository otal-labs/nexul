import { horizontalListSortingStrategy, SortableContext, useSortable } from "@dnd-kit/sortable";
import { ChevronsRightLeftIcon, PlusIcon } from "lucide-react";
import { memo, useMemo } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { boardGridTemplate, isFolded } from "@/components/board/boardGrid";
import { StatusMark } from "@/components/board/StatusIcon";
import type { DropTargetData } from "@/components/board/dragMove";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";
import { useBoardStore } from "@/stores/boardStore";

// The header's sortables carry this in place of a lane, so a stage drag only ever swaps with other header cells.
export const STAGE_HEADER = "__stage-header__";

interface BoardStageHeaderProps {
  columns: BoardStatus[];
  counts: Record<string, number>;
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

interface StageHeaderCellProps {
  column: BoardStatus;
  count: number;
  folded: boolean;
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

// Memoized: a refetch rebuilds the counts object, but a cell re-renders only when its own stage, count or fold changes.
const StageHeaderCell = memo(({ column, count, folded, onAddTicket }: StageHeaderCellProps) => {
  const draggable = useActionState({ permission: "projects:write" }).state === "live";
  const toggleDone = useBoardStore((s) => s.toggleDone);
  const { setNodeRef, setActivatorNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: `column-header-${column.id}`,
    disabled: { draggable: !draggable },
    data: { type: "column", statusId: column.id, categoryId: STAGE_HEADER, kind: column.kind } satisfies DropTargetData,
  });
  const style = {
    transform: transform ? `translate3d(${Math.round(transform.x)}px, 0, 0)` : undefined,
    transition,
  };

  return (
    <>
      {folded && (
      <div ref={setNodeRef} style={style} className="flex justify-center">
        <button
          type="button"
          onClick={toggleDone}
          aria-label={`Show ${column.name}, ${count} tickets`}
          title={`${column.name}: ${count}`}
          className="flex h-8 w-full items-center justify-center gap-1 rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground"
        >
          <StatusMark status={column} className="size-3.5" />
          <span className="font-mono text-xs tabular-nums">{count}</span>
        </button>
      </div>
      )}
      {!folded && (
    <h4
      ref={setNodeRef}
      style={style}
      className={cn("flex h-8 min-w-0 items-center gap-1.5 px-1.5", isDragging && "relative z-10 rounded-md bg-card shadow-elevated")}
    >
      <span
        ref={setActivatorNodeRef}
        {...(draggable ? { ...attributes, ...listeners, "aria-label": `Reorder ${column.name}` } : {})}
        className={cn(
          "flex min-w-0 flex-1 items-center gap-1.5 rounded-md",
          draggable && "cursor-grab select-none focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none active:cursor-grabbing",
        )}
      >
        <StatusMark status={column} className="size-3.5" />
        <span className="truncate text-[13px] font-semibold">{column.name}</span>
        <span className="font-mono text-xs tabular-nums text-muted-foreground" aria-label={`${count} tickets`}>
          {count}
        </span>
      </span>
      {column.kind === "done" && (
        <button
          type="button"
          onClick={toggleDone}
          aria-label={`Fold ${column.name}`}
          title={`Fold ${column.name}`}
          className="flex size-6 items-center justify-center rounded-md text-muted-foreground/60 transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground"
        >
          <ChevronsRightLeftIcon className="size-3.5" aria-hidden />
        </button>
      )}
      <ActionButton
        permission="tickets:write"
        verb="create tickets"
        variant="ghost"
        size="icon"
        icon={PlusIcon}
        onClick={() => onAddTicket(null, column.id)}
        aria-label={`New ticket in ${column.name}`}
        className="size-6 text-muted-foreground/60"
      />
    </h4>
      )}
    </>
  );
});

// The stage row, drawn once above every lane and held at the top of the board's scroll.
export const BoardStageHeader = ({ columns, counts, onAddTicket }: BoardStageHeaderProps) => {
  const doneOpen = useBoardStore((s) => s.doneOpen);
  const ids = useMemo(() => columns.map((column) => `column-header-${column.id}`), [columns]);
  return (
    <div className="sticky top-0 z-20 rounded-md bg-[var(--panel)] pb-1">
      <SortableContext items={ids} strategy={horizontalListSortingStrategy}>
        <div className="grid gap-2" style={{ gridTemplateColumns: boardGridTemplate(columns, doneOpen) }}>
          {columns.map((column) => (
            <StageHeaderCell
              key={column.id}
              column={column}
              count={counts[column.id] ?? 0}
              folded={isFolded(column, doneOpen)}
              onAddTicket={onAddTicket}
            />
          ))}
        </div>
      </SortableContext>
    </div>
  );
};

