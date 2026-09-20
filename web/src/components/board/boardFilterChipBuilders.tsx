import type { ChipOption } from "@/components/board/FilterChipRow";
import type { Category } from "@/models/Category";
import type { Project } from "@/models/Project";
import type { BoardStatus } from "@/models/Status";
import type { TicketType } from "@/models/TicketType";

export interface BoardFilters {
  projectIds: string[];
  categoryId: string | null;
  labels: string[];
  typeId: string | null;
  statusIds: string[];
  assignees: string[];
}

export interface FilterRow {
  title: string;
  chips: ChipOption[];
}

export const countActiveFilters = (filters: BoardFilters, hideProjectFilter: boolean): number =>
  (hideProjectFilter ? 0 : filters.projectIds.length) +
  (filters.categoryId ? 1 : 0) +
  filters.labels.length +
  (filters.typeId ? 1 : 0) +
  filters.statusIds.length +
  filters.assignees.length;

// "No projects yet" earns its keep; the zero-filter default otherwise needs no summary line.
export const buildSummaryLabel = (activeCount: number, projects: Project[] | undefined): string | null =>
  activeCount === 0 && (projects?.length ?? 0) === 0 ? "No projects yet" : null;

export const buildProjectChips = (projects: Project[] | undefined, filters: BoardFilters, onToggle: (id: string) => void): ChipOption[] =>
  (projects ?? []).map((project) => ({
    key: project.id,
    label: project.name,
    active: filters.projectIds.includes(project.id),
    ariaLabel: `Filter by ${project.name}`,
    onToggle: () => onToggle(project.id),
  }));

export const buildCategoryChips = (categories: Category[] | undefined, filters: BoardFilters, onToggle: (id: string | null) => void): ChipOption[] => {
  const list = categories ?? [];
  if (list.length === 0) return [];
  return [
    {
      key: "__uncategorized__",
      label: "Uncategorized",
      active: filters.categoryId === "uncategorized",
      onToggle: () => onToggle(filters.categoryId === "uncategorized" ? null : "uncategorized"),
    },
    ...list.map((category) => ({
      key: category.id,
      label: category.name,
      active: filters.categoryId === category.id,
      onToggle: () => onToggle(filters.categoryId === category.id ? null : category.id),
    })),
  ];
};

export const buildLabelChips = (labels: string[] | undefined, filters: BoardFilters, onToggle: (label: string) => void): ChipOption[] =>
  (labels ?? []).map((label) => ({
    key: label,
    label,
    active: filters.labels.includes(label),
    onToggle: () => onToggle(label),
  }));

export const buildTypeChips = (ticketTypes: TicketType[] | undefined, filters: BoardFilters, onSelectType: (id: string | null) => void): ChipOption[] =>
  (ticketTypes ?? []).map((type) => ({
    key: type.id,
    label: type.name,
    active: filters.typeId === type.id,
    onToggle: () => onSelectType(filters.typeId === type.id ? null : type.id),
  }));

export const buildStatusChips = (statuses: BoardStatus[] | undefined, filters: BoardFilters, onToggle: (id: string) => void): ChipOption[] =>
  (statuses ?? []).map((status) => ({
    key: status.id,
    label: status.name,
    active: filters.statusIds.includes(status.id),
    onToggle: () => onToggle(status.id),
  }));

export const buildFilterRows = (params: {
  hideProjectFilter: boolean;
  projects: Project[] | undefined;
  projectChips: ChipOption[];
  categoryChips: ChipOption[];
  labelChips: ChipOption[];
  typeChips: ChipOption[];
  statusChips: ChipOption[];
}): FilterRow[] => {
  const rows: FilterRow[] = [];
  const showProjects = !params.hideProjectFilter && (params.projects?.length ?? 0) > 0;
  if (showProjects) rows.push({ title: "Projects", chips: params.projectChips });
  if (params.categoryChips.length > 0) rows.push({ title: "Category", chips: params.categoryChips });
  if (params.labelChips.length > 0) rows.push({ title: "Labels", chips: params.labelChips });
  if (params.typeChips.length > 0) rows.push({ title: "Type", chips: params.typeChips });
  if (params.statusChips.length > 0) rows.push({ title: "Status", chips: params.statusChips });
  return rows;
};
