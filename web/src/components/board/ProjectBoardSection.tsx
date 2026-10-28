import { ticketDeveloper, ticketTester, type TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { BoardFilterBar, type BoardFilters } from "@/components/board/BoardFilterBar";
import type { DragMoveAction } from "@/components/board/dragMove";
import { KanbanBoard, type Swimlane } from "@/components/board/KanbanBoard";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import type { Project } from "@/models/Project";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

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
  const ticketCount = projectTickets?.length;
  const people = new Set(projectTickets?.flatMap((t) => [ticketDeveloper(t).key, ticketTester(t).key]).filter(Boolean)).size;
  return (
    <>
      {/* Title, count and toolbar share one row from 1280px: the board is a work surface, so the cards start high on the page. */}
      <div className="flex shrink-0 flex-col gap-3 xl:flex-row xl:items-center">
      <PageHeader
        className="shrink-0 border-b-0 pb-0"
        title={
          <div className="flex min-w-0 items-baseline gap-3">
            <h1 dir="auto" className="truncate text-base font-semibold tracking-[-0.011em]">
              {project?.name ?? "Board"}
            </h1>
            {ticketCount !== undefined && (
              <p className="shrink-0 text-xs tabular-nums text-muted-foreground">
                {ticketCount} {ticketCount === 1 ? "ticket" : "tickets"}
                {people > 0 && ` · ${people} ${people === 1 ? "person" : "people"}`}
              </p>
            )}
          </div>
        }
      />
      {tickets && (
        // The board is always project-scoped (ticket 08), so the multi-project filter row would always be redundant with the URL.
        <div className="min-w-0 flex-1">
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
