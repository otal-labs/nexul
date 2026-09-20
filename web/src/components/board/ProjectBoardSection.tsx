import { BoardFilterBar, type BoardFilters } from "@/components/board/BoardFilterBar";
import type { DragMoveAction } from "@/components/board/dragMove";
import { KanbanBoard, type Swimlane } from "@/components/board/KanbanBoard";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

interface ProjectBoardSectionProps {
  projectId?: string | undefined;
  projectName?: string | undefined;
  isLoading: boolean;
  error: unknown;
  tickets: Ticket[] | undefined;
  swimlanes: Swimlane[];
  columns: BoardStatus[];
  filters: BoardFilters;
  assignees: string[];
  onToggleProject: (projectId: string) => void;
  onToggleCategory: (categoryId: string | null) => void;
  onToggleLabel: (label: string) => void;
  onSelectType: (typeId: string | null) => void;
  onToggleStatus: (statusId: string) => void;
  onToggleAssignee: (assignee: string) => void;
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
  projectName,
  isLoading,
  error,
  tickets,
  swimlanes,
  columns,
  filters,
  assignees,
  onToggleProject,
  onToggleCategory,
  onToggleLabel,
  onSelectType,
  onToggleStatus,
  onToggleAssignee,
  onClear,
  onNewTicket,
  onNewCategory,
  onDrop,
  onReorderColumns,
  onAddTicket,
}: ProjectBoardSectionProps) => (
  <>
    <PageHeader title={projectName ?? "Board"} subtitle="Every ticket in its lane, traffic optional." />
    {isLoading && <LoadingDisplay label="Loading board…" />}
    {error && <ErrorDisplay error={error} title="Failed to load the board." />}
    {tickets && (
      // The board is always project-scoped (ticket 08), so the multi-project filter row would always be redundant with the URL.
      <BoardFilterBar
        projectId={projectId}
        hideProjectFilter
        assignees={assignees}
        filters={filters}
        onToggleProject={onToggleProject}
        onToggleCategory={onToggleCategory}
        onToggleLabel={onToggleLabel}
        onSelectType={onSelectType}
        onToggleStatus={onToggleStatus}
        onToggleAssignee={onToggleAssignee}
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
          filters.labels.length + filters.statusIds.length + filters.assignees.length > 0
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
