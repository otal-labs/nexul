import { ticketDeveloper, ticketTester, type TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { BoardFilterBar, type BoardFilters } from "@/components/board/BoardFilterBar";
import { BoardStageProgress } from "@/components/board/BoardStageProgress";
import { boardGridMaxWidth, countByStatus } from "@/components/board/boardGrid";
import { useState } from "react";
import type { DragMoveAction } from "@/components/board/dragMove";
import { KanbanBoard, type Swimlane } from "@/components/board/KanbanBoard";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import { ProjectMark } from "@/components/project/ProjectMark";
import type { Project } from "@/models/Project";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";
import { useBoardStore } from "@/stores/boardStore";

interface ProjectBoardSectionProps {
  projectId?: string | undefined;
  project?: Project | undefined;
  isLoading: boolean;
  error: unknown;
  tickets: Ticket[] | undefined;
  swimlanes: Swimlane[];
  columns: BoardStatus[];
  filters: BoardFilters;
  developers: TicketPersonRef[];
  showWaitingForMeToTest: boolean;
  showPlaying: boolean;
  onToggleProject: (projectId: string) => void;
  onToggleCategory: (categoryId: string | null) => void;
  onToggleLabel: (label: string) => void;
  onSelectType: (typeId: string | null) => void;
  onToggleStatus: (statusId: string) => void;
  onToggleDeveloper: (developer: string) => void;
  onToggleWaitingForMeToTest: () => void;
  onTogglePlaying: () => void;
  onSearch: (search: string) => void;
  onClear: () => void;
  onNewTicket: () => void;
  onNewCategory: () => void;
  onDrop: (actions: DragMoveAction[]) => Promise<void> | void;
  onReorderColumns: (statusIds: string[]) => Promise<void> | void;
  onAddTicket: (categoryId: string | null, statusId: string) => void;
}

// Project-scoped title + filter bar + kanban board; BoardPage keeps only fetching and the no-project gate.
export const ProjectBoardSection = ({
  projectId,
  project,
  isLoading,
  error,
  tickets,
  swimlanes,
  columns,
  filters,
  developers,
  showWaitingForMeToTest,
  showPlaying,
  onToggleProject,
  onToggleCategory,
  onToggleLabel,
  onSelectType,
  onToggleStatus,
  onToggleDeveloper,
  onToggleWaitingForMeToTest,
  onTogglePlaying,
  onSearch,
  onClear,
  onNewTicket,
  onNewCategory,
  onDrop,
  onReorderColumns,
  onAddTicket,
}: ProjectBoardSectionProps) => {
  const projectTickets = tickets?.filter((t) => t.project_id === projectId);
  const shown = swimlanes.flatMap((lane) => lane.tickets);
  const shownCount = shown.length;
  const totalCount = projectTickets?.length ?? 0;
  const people = new Set(shown.flatMap((t) => [ticketDeveloper(t).key, ticketTester(t).key]).filter(Boolean)).size;
  const stageCounts = countByStatus(swimlanes);
  const doneOpen = useBoardStore((s) => s.doneOpen);
  const [gutter, setGutter] = useState(0);
  // The chrome never scrolls with the grid, so it takes the grid's laid-out width when that is narrower than the panel, or the toolbar's own floor when the grid is smaller still.
  const chromeWidth = `max(45.625rem, ${boardGridMaxWidth(columns, doneOpen)})`;
  const countLine =
    shownCount === totalCount ? `${totalCount} ${totalCount === 1 ? "ticket" : "tickets"}` : `${shownCount} of ${totalCount} tickets`;
  return (
    <>
      <div className="mx-auto flex w-full shrink-0 flex-col" style={{ maxWidth: chromeWidth, paddingRight: gutter || undefined }}>
        <PageHeader
          className="border-b-0 pb-0"
          leading={project ? <ProjectMark project={project} className="size-8 rounded-lg text-[11px]" /> : undefined}
          title={project?.name ?? "Board"}
          meta={
            <p className="font-mono text-xs tabular-nums">
              {countLine}
              {people > 0 && ` · ${people} ${people === 1 ? "person" : "people"}`}
            </p>
          }
          actions={<BoardStageProgress columns={columns} counts={stageCounts} />}
          actionsClassName="self-center"
        />
        {tickets && (
          // The board is always project-scoped (ticket 08), so the multi-project filter row would always be redundant with the URL.
          <div className="mt-3">
            <BoardFilterBar
              projectId={projectId}
              hideProjectFilter
              developers={developers}
              showWaitingForMeToTest={showWaitingForMeToTest}
              showPlaying={showPlaying}
              filters={filters}
              onToggleProject={onToggleProject}
              onToggleCategory={onToggleCategory}
              onToggleLabel={onToggleLabel}
              onSelectType={onSelectType}
              onToggleStatus={onToggleStatus}
              onToggleDeveloper={onToggleDeveloper}
              onToggleWaitingForMeToTest={onToggleWaitingForMeToTest}
              onTogglePlaying={onTogglePlaying}
              onSearch={onSearch}
              onClear={onClear}
              onNewTicket={onNewTicket}
              onNewCategory={onNewCategory}
            />
          </div>
        )}
      </div>
      {isLoading && <LoadingDisplay label="Loading board…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load the board." />}
      {tickets && swimlanes.length === 0 && (
        <NoDataDisplay
          message={
            // projectIds is the route's scope here, not a user-picked filter (same rule as BoardFilterBar's count).
            filters.categoryId ||
            filters.typeId ||
            filters.waitingForMeToTest ||
            filters.search.trim() !== "" ||
            filters.labels.length + filters.statusIds.length + filters.developers.length > 0
              ? "No tickets match these filters."
              : "No tickets yet. Create one to start the board."
          }
        />
      )}
      {tickets && swimlanes.length > 0 && (
        <KanbanBoard
          projectId={projectId}
          maxWidth={chromeWidth}
          onGutter={setGutter}
          columns={columns}
          swimlanes={swimlanes}
          onDrop={onDrop}
          onReorderColumns={onReorderColumns}
          onAddTicket={onAddTicket}
        />
      )}
    </>
  );
};
