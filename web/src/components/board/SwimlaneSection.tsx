import { useDroppable } from "@dnd-kit/core";
import { horizontalListSortingStrategy, SortableContext } from "@dnd-kit/sortable";
import { ChevronDownIcon, CircleCheckBig, CircleHelp, LoaderCircle } from "lucide-react";
import { useParams } from "react-router";

import { KanbanColumn } from "@/components/board/KanbanColumn";
import type { Swimlane } from "@/components/board/KanbanBoard";
import type { DropTargetData } from "@/components/board/dragMove";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useTicketRunCounts } from "@/hooks/TrailHooks";
import { cn } from "@/lib/utils";
import { resolveProject } from "@/models/Project";
import type { BoardStatus } from "@/models/Status";
import { useBoardStore } from "@/stores/boardStore";

interface SwimlaneSectionProps {
  lane: Swimlane;
  columns: BoardStatus[];
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

export const SwimlaneSection = ({ lane, columns, onAddTicket }: SwimlaneSectionProps) => {
  const { projectId: routeParam = "" } = useParams();
  const { data: projects = [] } = useFetchProjects();
  const projectId = resolveProject(projects, routeParam)?.id ?? routeParam;
  const collapsed = useBoardStore((s) => s.collapsedLanes[projectId]?.includes(lane.key) ?? false);
  const toggleLane = useBoardStore((s) => s.toggleLane);
  const runs = useTicketRunCounts(lane.tickets[0]?.project_id, lane.tickets.map((t) => t.id));
  const doneStatusIds = new Set(columns.filter((c) => c.kind === "done").map((c) => c.id));
  const allDone = lane.tickets.length > 0 && lane.tickets.every((t) => doneStatusIds.has(t.status));
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
      <h3 className="border-b border-border pb-1">
        {/* The whole header row toggles collapse; label and count stay sticky against the horizontal scroll. */}
        <button
          type="button"
          aria-expanded={!collapsed}
          onClick={() => toggleLane(projectId, lane.key)}
          className="group/lane flex w-full cursor-pointer items-center justify-between gap-2 rounded-sm px-1.5 py-1 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
        >
          <span className="sticky left-0 flex min-w-0 items-center gap-3">
            <span className="min-w-0 truncate text-sm font-semibold">{lane.label}</span>{" "}
            {allDone && <CircleCheckBig className="size-3 shrink-0 text-success" role="img" aria-label="All done" />}{" "}
            {runs.running + runs.waiting > 0 && (
              <span className="flex shrink-0 items-center gap-2.5 font-mono text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <LoaderCircle className="size-3 animate-spin text-warning motion-reduce:animate-none" role="img" aria-label="Running" />{" "}
                  {runs.running} {runs.running === 1 ? "ticket" : "tickets"}
                </span>{" "}
                <span className="flex items-center gap-1">
                  <CircleHelp className="size-3 text-info" role="img" aria-label="Waiting for an answer" />{" "}
                  {runs.waiting} {runs.waiting === 1 ? "ticket" : "tickets"}
                </span>
              </span>
            )}
          </span>{" "}
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
