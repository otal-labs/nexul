import { useMemo } from "react";

import type { Swimlane } from "@/components/board/KanbanBoard";
import type { BoardFilters } from "@/components/board/BoardFilterBar";
import type { Category } from "@/models/Category";
import type { Ticket } from "@/models/Ticket";

// Every category gets a lane up front, even empty ones, or a new lane can't be dragged into before its first ticket.
const seedCategoryLanes = (categories: Category[], filters: BoardFilters): Map<string, Swimlane> => {
  const byName = new Map<string, Swimlane>();
  for (const category of categories) {
    if (filters.projectIds.length > 0 && !filters.projectIds.includes(category.project_id)) continue;
    if (filters.categoryId === "uncategorized") continue;
    if (filters.categoryId && filters.categoryId !== category.id) continue;
    if (byName.has(category.name)) continue;
    byName.set(category.name, { key: category.name, label: category.name, categoryId: category.id, tickets: [] });
  }
  return byName;
};

const assignTicketToLane = (byName: Map<string, Swimlane>, ticket: Ticket, categories: Category[]) => {
  const category = ticket.category_id ? categories.find((c) => c.id === ticket.category_id) : undefined;
  const name = category?.name;
  const key = name ?? "__uncategorized__";
  const lane = byName.get(key);
  if (lane) {
    lane.tickets.push(ticket);
    return;
  }
  byName.set(key, { key, label: name ?? "Uncategorized", categoryId: category?.id ?? null, tickets: [ticket] });
};

// Groups tickets by category name; same-named lanes merge across selected projects, uncategorized gets its own.
export const useSwimlanes = (
  filteredTickets: Ticket[],
  categories: Category[],
  filters: BoardFilters,
): Swimlane[] =>
  useMemo(() => {
    const byName = seedCategoryLanes(categories, filters);
    for (const ticket of filteredTickets) assignTicketToLane(byName, ticket, categories);
    return Array.from(byName.values());
  }, [filteredTickets, categories, filters]);
