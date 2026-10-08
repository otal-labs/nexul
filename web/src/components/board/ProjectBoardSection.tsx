import { BoardFilterBar, type BoardFilters } from "@/components/board/BoardFilterBar";
import type { DragMoveAction } from "@/components/board/dragMove";
import { KanbanBoard, type Swimlane } from "@/components/board/KanbanBoard";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { BoardStageSummary } from "@/components/board/BoardStageSummary";
import { PageHeader, pageTitleClass } from "@/components/PageHeader";
import { ProjectMark } from "@/components/project/ProjectMark";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";
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
  developers: string[];
  showWaitingForMeToTest: boolean;
  onToggleProject: (projectId: string) => void;
  onToggleCategory: (categoryId: string | null) => void;
  onToggleLabel: (label: string) => void;
  onSelectType: (typeId: string | null) => void;
  onToggleStatus: (statusId: string) => void;
  onToggleDeveloper: (developer: string) => void;
  onToggleWaitingForMeToTest: () => void;
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
  onToggleProject,
  onToggleCategory,
  onToggleLabel,
  onSelectType,
  onToggleStatus,
  onToggleDeveloper,
  onToggleWaitingForMeToTest,
  onSearch,
  onClear,
  onNewTicket,
  onNewCategory,
  onDrop,
  onReorderColumns,
  onAddTicket,
}: ProjectBoardSectionProps) => {
  const workspaceCrumb = useWorkspaceCrumb();
  const projectTickets = tickets?.filter((t) => t.project_id === projectId);
  const ticketCount = projectTickets?.length;
  const people = new Set(projectTickets?.flatMap((t) => [t.developer, t.tester]).filter(Boolean)).size;
  return (
    <>
      <PageHeader
        crumbs={[workspaceCrumb]}
        title={
          <div className="flex items-center gap-3.5">
            {project && <ProjectMark project={project} />}
            <div className="min-w-0">
              <h1 className={pageTitleClass}>{project?.name ?? "Board"}</h1>
              {ticketCount !== undefined && (
                <p className="mt-1 font-mono text-xs tabular-nums text-muted-foreground">
                  {ticketCount} {ticketCount === 1 ? "ticket" : "tickets"}
                  {people > 0 && ` · ${people} ${people === 1 ? "person" : "people"}`}
                </p>
              )}
            </div>
          </div>
        }
        actions={<BoardStageSummary projectId={projectId} className="w-96" />}
      />
      {isLoading && <LoadingDisplay label="Loading board…" />}
      {error && <ErrorDisplay error={error} title="Failed to load the board." />}
      {tickets && (
        // The board is always project-scoped (ticket 08), so the multi-project filter row would always be redundant with the URL.
        <BoardFilterBar
          projectId={projectId}
          hideProjectFilter
          developers={developers}
          showWaitingForMeToTest={showWaitingForMeToTest}
          filters={filters}
          onToggleProject={onToggleProject}
          onToggleCategory={onToggleCategory}
          onToggleLabel={onToggleLabel}
          onSelectType={onSelectType}
          onToggleStatus={onToggleStatus}
          onToggleDeveloper={onToggleDeveloper}
          onToggleWaitingForMeToTest={onToggleWaitingForMeToTest}
          onSearch={onSearch}
          onClear={onClear}
          onNewTicket={onNewTicket}
          onNewCategory={onNewCategory}
        />
      )}
      {tickets && swimlanes.length === 0 && (
        <NoDataDisplay
          message={
            // projectIds is the route's scope here, not a user-picked filter (same rule as BoardFilterBar's count).
            filters.categoryId ||
            filters.typeId ||
            filters.waitingForMeToTest ||
            filters.search.trim() !== "" ||
            filters.labels.length + filters.statusIds.length + filters.developers.length > 0
              ? "No tickets match the active filters."
              : "No tickets yet — create the first one."
          }
        />
      )}
      {tickets && swimlanes.length > 0 && (
        <KanbanBoard
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
