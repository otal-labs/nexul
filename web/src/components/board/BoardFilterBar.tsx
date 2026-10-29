import { FilterIcon, FlaskConicalIcon, SearchIcon, SparklesIcon } from "lucide-react";

import type { TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { BoardCreateMenu } from "@/components/board/BoardCreateMenu";
import { BoardFilterMenu } from "@/components/board/BoardFilterMenu";
import { FilterChipRow } from "@/components/board/FilterChipRow";
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
import { Input } from "@/components/ui/input";
import { Popover, PopoverTrigger } from "@/components/ui/popover";
import { useFetchProjectCategories } from "@/hooks/CategoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useFetchAllLabels, useFetchLabelColors } from "@/hooks/TicketHooks";
import { useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { cn } from "@/lib/utils";

export type { BoardFilters } from "@/components/board/boardFilterChipBuilders";

interface BoardFilterBarProps {
  /** Resolved project id for the scoped board — the route param itself can be a prefix, so callers resolve it. */
  projectId?: string | undefined;
  /** Hides the Projects filter row/toggle on a /board/:projectId view, where the URL already scopes to one project. */
  hideProjectFilter?: boolean;
  /** Derived client-side from the page's own ticket set (useBoardFilters); shown as the avatar stack, not as popover chips. */
  developers: TicketPersonRef[];
  showWaitingForMeToTest: boolean;
  showPlaying: boolean;
  filters: BoardFilters;
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
}

// Reference data is fetched here, not drilled down; each hook shares its cache, so this adds no requests.
export const BoardFilterBar = ({
  projectId,
  hideProjectFilter = false,
  developers,
  showWaitingForMeToTest,
  showPlaying,
  filters,
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
}: BoardFilterBarProps) => {
  const { data: projects } = useFetchProjects();
  const { data: categories } = useFetchProjectCategories(projectId);
  const { data: labels } = useFetchAllLabels(projectId);
  const { data: labelColors } = useFetchLabelColors(projectId);
  const { data: ticketTypes } = useFetchProjectTicketTypes(projectId);
  const { data: statuses } = useFetchProjectStatuses(projectId);

  const activeCount = countActiveFilters(filters, hideProjectFilter);
  const projectChips = buildProjectChips(projects, filters, onToggleProject);
  const categoryChips = buildCategoryChips(categories, filters, onToggleCategory);
  const labelChips = buildLabelChips(labels, labelColors, filters, onToggleLabel);
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
  const activeChips = filterRows.flatMap((row) => row.chips.filter((option) => option.active));

  return (
    <div className="@container">
      {/* The text filter leads the filters it works with; Create alone holds the right edge. */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 *:shrink-0">
          <div className="relative min-w-[12.5rem] max-w-[22.5rem] grow basis-60">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" aria-hidden />
            <Input
              type="search"
              aria-label="Filter tickets"
              placeholder="Filter tickets"
              value={filters.search}
              onChange={(e) => onSearch(e.target.value)}
              className="h-8 pl-8 text-[13px]"
            />
          </div>
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" size="sm" className="h-8 px-2.5 text-xs">
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
              title="Mine to test"
              className={cn("h-8 px-2.5 text-xs", !filters.waitingForMeToTest && "text-muted-foreground")}
            >
              <FlaskConicalIcon className="size-3.5" />
              {/* Icon-only below 1060, named by its tooltip, so the toolbar keeps one row under the title. */}
              <span className="max-[1060px]:sr-only">Mine to test</span>
            </Button>
          )}
          {showPlaying && (
            <Button
              variant={filters.playing ? "default" : "outline"}
              size="sm"
              aria-pressed={filters.playing}
              onClick={onTogglePlaying}
              title="Playing"
              className={cn("h-8 px-2.5 text-xs", !filters.playing && "text-muted-foreground")}
            >
              <SparklesIcon className="size-3.5" />
              <span className="max-[1060px]:sr-only">Playing</span>
            </Button>
          )}
          {summaryLabel && <span className="whitespace-nowrap text-xs text-muted-foreground">{summaryLabel}</span>}
        </div>
        <div className="ml-auto flex shrink-0 items-center">
          <BoardCreateMenu projectId={projectId} onNewTicket={onNewTicket} onNewCategory={onNewCategory} />
        </div>
      </div>
      {activeChips.length > 0 && (
        <div className="mt-2">
          <FilterChipRow chips={activeChips} onClearAll={onClear} />
        </div>
      )}
    </div>
  );
};
