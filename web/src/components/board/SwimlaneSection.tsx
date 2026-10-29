import { useDroppable } from "@dnd-kit/core";
import { ChevronDownIcon, CircleCheckBig, CircleHelp, LoaderCircle } from "lucide-react";
import { useParams } from "react-router";

import { boardGridTemplate, isFolded } from "@/components/board/boardGrid";
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
  /** There is more board to the right: fade each lane's edge away rather than cutting it. */
  overflowing?: boolean;
}

export const SwimlaneSection = ({ lane, columns, overflowing = false }: SwimlaneSectionProps) => {
  const { projectId: routeParam = "" } = useParams();
  const { data: projects = [] } = useFetchProjects();
  const projectId = resolveProject(projects, routeParam)?.id ?? routeParam;
  const collapsed = useBoardStore((s) => s.collapsedLanes[projectId]?.includes(lane.key) ?? false);
  const toggleLane = useBoardStore((s) => s.toggleLane);
  const doneOpen = useBoardStore((s) => s.doneOpen);
  const runs = useTicketRunCounts(lane.tickets[0]?.project_id, lane.tickets.map((t) => t.id));
  const doneStatusIds = new Set(columns.filter((c) => c.kind === "done").map((c) => c.id));
  const doneCount = lane.tickets.filter((t) => doneStatusIds.has(t.status)).length;
  const allDone = lane.tickets.length > 0 && doneCount === lane.tickets.length;
  const { setNodeRef, isOver } = useDroppable(
    lane.categoryId === null
      ? { id: `lane-${lane.key}`, disabled: true }
      : { id: `lane-${lane.key}`, data: { type: "lane", categoryId: lane.categoryId } satisfies DropTargetData },
  );

  return (
    <section
      ref={setNodeRef}
      aria-label={`${lane.label} swimlane`}
      // A band across the stage columns, then this lane's cells on the board's shared grid.
      className={cn(
        "space-y-1.5 rounded-md transition-[box-shadow] duration-150 ease-standard",
        isOver && "ring-1 ring-inset ring-ring",
      )}
    >
      {/* Pinned under the stage header while its lane scrolls; label and count also stay put against the horizontal scroll. */}
      <h3 className="sticky top-9 z-10 rounded-md border-b border-border bg-[var(--panel)]">
        <button
          type="button"
          aria-expanded={!collapsed}
          onClick={() => toggleLane(projectId, lane.key)}
          className="group/lane flex h-8 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left transition-colors duration-150 ease-standard hover:bg-accent"
        >
          {/* Inside the visible strip, so a long name truncates before the progress cluster instead of running under it. */}
          <span className="sticky left-2 flex min-w-0 max-w-[calc(100cqw-15rem)] flex-1 items-center gap-2">
            <ChevronDownIcon
              className={cn("size-3.5 shrink-0 self-center text-muted-foreground transition-transform duration-150 ease-standard group-hover/lane:text-foreground", collapsed && "-rotate-90")}
              aria-hidden
            />
            <span title={lane.label} className="min-w-0 truncate text-[14px] font-semibold">{lane.label}</span>{" "}
            {allDone && <CircleCheckBig className="size-3 shrink-0 self-center text-success" role="img" aria-label="All done" />}{" "}
            {runs.running + runs.waiting > 0 && (
              <span className="flex shrink-0 items-baseline gap-2.5 text-xs tabular-nums text-muted-foreground">
                {runs.running > 0 && (
                  <span className="flex items-baseline gap-1">
                    <LoaderCircle className="size-3 self-center animate-spin text-warning motion-reduce:animate-none" role="img" aria-label="Running" />{" "}
                    {runs.running} {runs.running === 1 ? "ticket" : "tickets"}
                  </span>
                )}{" "}
                {runs.waiting > 0 && (
                  <span className="flex items-baseline gap-1">
                    <CircleHelp className="size-3 self-center text-info" role="img" aria-label="Waiting for an answer" />{" "}
                    {runs.waiting} {runs.waiting === 1 ? "ticket" : "tickets"}
                  </span>
                )}
              </span>
            )}
          </span>{" "}
          {/* The lane's own progress, held at the right edge of what the board shows, so it never scrolls away with the last stage. */}
          <span className="sticky right-2 ml-auto flex w-[200px] shrink-0 items-center justify-end gap-2">
            <span className="h-1 w-24 shrink-0 overflow-hidden rounded-full bg-muted max-md:hidden" aria-hidden>
              <span className="block h-full rounded-full bg-success" style={{ width: `${lane.tickets.length === 0 ? 0 : Math.round((doneCount / lane.tickets.length) * 100)}%` }} />
            </span>
            {/* The count keeps its 72px whether or not the lane has tickets, so the line does not slide between lanes. */}
            <span className="min-w-[72px] shrink-0 text-right font-mono text-[11px] tabular-nums text-muted-foreground">
              {lane.tickets.length > 0 ? `${doneCount}/${lane.tickets.length} done` : ""}
            </span>
          </span>
        </button>
      </h3>
      {/* Content stays mounted (inert) so expanding never replays entrances; inline-size containment drops a collapsed lane's columns from the shared scroll width. */}
      <div
        inert={collapsed}
        aria-hidden={collapsed}
        data-closed={collapsed || undefined}
        className={cn("disclosure", collapsed && "contain-inline-size")}
      >
        <div className="min-h-0 overflow-hidden">
          <div
            data-board-grid="lane"
            className={cn("grid gap-2", overflowing && "[mask-image:linear-gradient(to_right,#000_calc(100%-1rem),transparent)]")}
            style={{ gridTemplateColumns: boardGridTemplate(columns, doneOpen) }}
          >
              {columns.map((column) => (
                <KanbanColumn
                  key={`${lane.key}-${column.id}`}
                  droppableId={`column-${lane.key}-${column.id}`}
                  laneLabel={lane.label}
                  categoryId={lane.categoryId ?? ""}
                  column={column}
                  collapsed={collapsed}
                  folded={isFolded(column, doneOpen)}
                  tickets={lane.tickets.filter((t) => t.status === column.id).sort((a, b) => a.position - b.position)}
                />
              ))}
          </div>
        </div>
      </div>
    </section>
  );
};
