import { useParams } from "react-router";

import { BoardNotFoundState } from "@/components/board/BoardNotFoundState";
import { BoardUnscopedStates } from "@/components/board/BoardUnscopedStates";
import { ProjectBoardSection } from "@/components/board/ProjectBoardSection";
import { Container } from "@/components/Container";
import { useFetchCategories } from "@/hooks/CategoryHooks";
import { useBoardActions } from "@/hooks/useBoardActions";
import { useBoardFilters } from "@/hooks/useBoardFilters";
import { useBoardProjectScope } from "@/hooks/useBoardProjectScope";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchTickets } from "@/hooks/TicketHooks";
import { useSwimlanes } from "@/hooks/useSwimlanes";

export const BoardPage = () => {
  // No coherent mixed-column board exists without :projectId — statuses/ticket_types are project-scoped (ticket 08).
  const { projectId: routeParam } = useParams();

  const { data: projects = [], error, isPending } = useFetchProjects();
  const { data: tickets, error: ticketsError, isPending: ticketsPending } = useFetchTickets();
  const { data: categories = [] } = useFetchCategories();

  const { scopedProject, scopedProjectId, scopedNotFound } = useBoardProjectScope({
    routeParam,
    projects,
    isLoading: isPending,
    error,
  });

  const { data: statuses = [] } = useFetchProjectStatuses(scopedProjectId);

  const {
    filters,
    assignees,
    filteredTickets,
    toggleProject,
    toggleCategory,
    toggleLabel,
    selectType,
    toggleStatus,
    toggleAssignee,
    clearAll,
  } = useBoardFilters(tickets, scopedProjectId);

  const swimlanes = useSwimlanes(filteredTickets, categories, filters);

  const {
    openCreateProjectDialog,
    openCreateTicketDialog,
    openCreateCategoryDialog,
    addTicketToColumn,
    dropTicket,
    reorderColumns,
  } = useBoardActions({ projects, selectedProjectIds: filters.projectIds, projectId: scopedProjectId });

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Full-bleed: the board is the one page where width is columns, so it gets the whole viewport. */}
      <Container className="max-w-none space-y-4 py-6">
        {!routeParam && (
          <BoardUnscopedStates
            isLoading={isPending}
            error={error}
            hasProjects={projects.length > 0}
            onCreateProject={() => void openCreateProjectDialog()}
          />
        )}
        {scopedNotFound && <BoardNotFoundState />}
        {routeParam && !scopedNotFound && (
          <ProjectBoardSection
            projectId={scopedProjectId}
            projectName={scopedProject?.name}
            isLoading={isPending || ticketsPending}
            error={error ?? ticketsError}
            tickets={tickets}
            swimlanes={swimlanes}
            columns={statuses}
            filters={filters}
            assignees={assignees}
            onToggleProject={toggleProject}
            onToggleCategory={toggleCategory}
            onToggleLabel={toggleLabel}
            onSelectType={selectType}
            onToggleStatus={toggleStatus}
            onToggleAssignee={toggleAssignee}
            onClear={clearAll}
            onNewTicket={() => void openCreateTicketDialog()}
            onNewCategory={() => void openCreateCategoryDialog()}
            onDrop={dropTicket}
            onReorderColumns={reorderColumns}
            onAddTicket={(categoryId, statusId) => void addTicketToColumn(categoryId, statusId)}
          />
        )}
      </Container>
    </div>
  );
};
