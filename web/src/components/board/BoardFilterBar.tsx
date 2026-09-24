import { FilterIcon, FlaskConicalIcon } from "lucide-react";

import { BoardCreateMenu } from "@/components/board/BoardCreateMenu";
import { BoardFilterMenu } from "@/components/board/BoardFilterMenu";
import {
  buildCategoryChips,
  buildFilterRows,
  buildLabelChips,
  buildProjectChips,
  buildStatusChips,
  buildSummaryLabel,
  buildTypeChips,
  countActiveFilters,
  type BoardFilters,
} from "@/components/board/boardFilterChipBuilders";
import { DeveloperStack } from "@/components/board/DeveloperStack";
import { Button } from "@/components/ui/button";
import { Popover, PopoverTrigger } from "@/components/ui/popover";
import { useFetchCategories } from "@/hooks/CategoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchAllLabels } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";

export type { BoardFilters } from "@/components/board/boardFilterChipBuilders";

interface BoardFilterBarProps {
  /** Resolved project id for the scoped board — the route param itself can be a prefix, so callers resolve it. */
  projectId?: string | undefined;
  /** Hides the Projects filter row/toggle on a /board/:projectId view, where the URL already scopes to one project. */
  hideProjectFilter?: boolean;
  /** Derived client-side from the page's own ticket set (useBoardFilters); shown as the avatar stack, not as popover chips. */
  developers: string[];
  showWaitingForMeToTest: boolean;
  filters: BoardFilters;
  onToggleProject: (projectId: string) => void;
  onToggleCategory: (categoryId: string | null) => void;
  onToggleLabel: (label: string) => void;
  onSelectType: (typeId: string | null) => void;
  onToggleStatus: (statusId: string) => void;
  onToggleDeveloper: (developer: string) => void;
  onToggleWaitingForMeToTest: () => void;
  onClear: () => void;
  onNewTicket: () => void;
  onNewCategory: () => void;
}

// Reference data is fetched here, not drilled down; each hook shares its cache, so this adds no requests.
export const BoardFilterBar = ({
  projectId,
  hideProjectFilter = false,
  developers,
  showWaitingForMeToTest,
  filters,
  onToggleProject,
  onToggleCategory,
  onToggleLabel,
  onSelectType,
  onToggleStatus,
  onToggleDeveloper,
  onToggleWaitingForMeToTest,
  onClear,
  onNewTicket,
  onNewCategory,
}: BoardFilterBarProps) => {
  const { data: projects } = useFetchProjects();
  const { data: categories } = useFetchCategories();
  const { data: labels } = useFetchAllLabels();
  const { data: ticketTypes } = useFetchProjectTicketTypes(projectId);
  const { data: statuses } = useFetchProjectStatuses(projectId);

  const activeCount = countActiveFilters(filters, hideProjectFilter);
  const projectChips = buildProjectChips(projects, filters, onToggleProject);
  const categoryChips = buildCategoryChips(categories, filters, onToggleCategory);
  const labelChips = buildLabelChips(labels, filters, onToggleLabel);
  const typeChips = buildTypeChips(ticketTypes, filters, onSelectType);
  const statusChips = buildStatusChips(statuses, filters, onToggleStatus);
  const summaryLabel = buildSummaryLabel(activeCount, projects);
  const filterRows = buildFilterRows({
    hideProjectFilter,
    projects,
    projectChips,
    categoryChips,
    labelChips,
    typeChips,
    statusChips,
  });

  return (
    <div className="animate-in fade-in-0 flex flex-wrap items-center gap-x-2 gap-y-2 rounded-md border border-border bg-card px-3 py-2 shadow-card duration-150 ease-out motion-reduce:animate-none">
      <Popover>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" className="h-9 px-3 text-xs">
            <FilterIcon className="size-3.5" />
            {activeCount > 0 ? `Filter (${activeCount})` : "Filter"}
          </Button>
        </PopoverTrigger>
        <BoardFilterMenu filterRows={filterRows} activeCount={activeCount} onClear={onClear} />
      </Popover>
      {developers.length > 0 && (
        <DeveloperStack developers={developers} selected={filters.developers} onToggle={onToggleDeveloper} />
      )}
      {showWaitingForMeToTest && (
        <Button
          variant={filters.waitingForMeToTest ? "default" : "outline"}
          size="sm"
          aria-pressed={filters.waitingForMeToTest}
          onClick={onToggleWaitingForMeToTest}
          className="h-9 px-3 text-xs"
        >
          <FlaskConicalIcon className="size-3.5" />
          Waiting for me to test
        </Button>
      )}
      {summaryLabel && <span className="whitespace-nowrap text-xs text-muted-foreground">{summaryLabel}</span>}
      <BoardCreateMenu onNewTicket={onNewTicket} onNewCategory={onNewCategory} className="ml-auto" />
    </div>
  );
};
