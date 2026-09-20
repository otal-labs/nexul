import { useDroppable } from "@dnd-kit/core";
import { horizontalListSortingStrategy, SortableContext } from "@dnd-kit/sortable";
import { ChevronDownIcon } from "lucide-react";

import { KanbanColumn } from "@/components/board/KanbanColumn";
import type { Swimlane } from "@/components/board/KanbanBoard";
import type { DropTargetData } from "@/components/board/dragMove";
import { cn } from "@/lib/utils";
import type { BoardStatus } from "@/models/Status";
import { useBoardStore } from "@/stores/boardStore";

interface SwimlaneSectionProps {
  lane: Swimlane;
  columns: BoardStatus[];
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

export const SwimlaneSection = ({ lane, columns, onAddTicket }: SwimlaneSectionProps) => {
  const collapsed = useBoardStore((s) => s.collapsedLaneKeys.includes(lane.key));
  const toggleLane = useBoardStore((s) => s.toggleLane);
  const { setNodeRef, isOver } = useDroppable(
    lane.categoryId === null
      ? { id: `lane-${lane.key}`, disabled: true }
      : { id: `lane-${lane.key}`, data: { type: "lane", categoryId: lane.categoryId } satisfies DropTargetData },
  );

  return (
    <section
      ref={setNodeRef}
      aria-label={`${lane.label} swimlane`}
      // Fixed-width columns overflow into the shared scroll container instead of wrapping; the lane header spans them all.
      className={cn(
        "w-max min-w-full space-y-2 rounded-md transition-[box-shadow] duration-150 ease-standard",
        isOver && "ring-1 ring-inset ring-ring",
      )}
    >
      <h3 className="border-b border-border pb-1.5">
        {/* The whole header row toggles collapse; label and count stay sticky against the horizontal scroll. */}
        <button
          type="button"
          aria-expanded={!collapsed}
          onClick={() => toggleLane(lane.key)}
          className="group/lane flex w-full cursor-pointer items-center justify-between gap-2 rounded-sm px-0.5 text-left"
        >
          <span className="sticky left-0 min-w-0 truncate text-sm font-semibold">{lane.label}</span>
          <span className="sticky right-0 flex shrink-0 items-center gap-1 font-mono text-xs text-muted-foreground transition-colors group-hover/lane:text-foreground">
            {lane.tickets.length} tickets
            <ChevronDownIcon
              className={cn("size-3.5 transition-transform duration-150 ease-standard", collapsed && "-rotate-90")}
              aria-hidden
            />
          </span>
        </button>
      </h3>
      {/* Grid-row 1fr→0fr collapse needs no measured height; content stays mounted (inert) so expanding never replays entrances. */}
      <div
        inert={collapsed}
        aria-hidden={collapsed}
        className={cn(
          "grid transition-[grid-template-rows,opacity] motion-reduce:transition-[opacity]",
          collapsed
            ? "[grid-template-rows:0fr] opacity-0 duration-150 ease-standard"
            : "[grid-template-rows:1fr] opacity-100 duration-200 ease-out",
        )}
      >
        <div className="min-h-0 overflow-hidden">
          <SortableContext
            items={columns.map((column) => `column-${lane.key}-${column.id}`)}
            strategy={horizontalListSortingStrategy}
          >
            <div className="flex gap-3">
              {columns.map((column) => (
                <KanbanColumn
                  key={`${lane.key}-${column.id}`}
                  droppableId={`column-${lane.key}-${column.id}`}
                  laneLabel={lane.label}
                  categoryId={lane.categoryId ?? ""}
                  column={column}
                  tickets={lane.tickets.filter((t) => t.status === column.id).sort((a, b) => a.position - b.position)}
                  onAddTicket={() => onAddTicket(lane.categoryId, column.id)}
                />
              ))}
            </div>
          </SortableContext>
        </div>
      </div>
    </section>
  );
};
