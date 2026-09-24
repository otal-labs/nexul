import { useEffect, useMemo, useState } from "react";

import type { BoardFilters } from "@/components/board/BoardFilterBar";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const EMPTY_FILTERS: BoardFilters = {
  projectIds: [],
  categoryId: null,
  labels: [],
  typeId: null,
  statusIds: [],
  developers: [],
  waitingForMeToTest: false,
};

const matchesCategory = (ticket: Ticket, categoryId: BoardFilters["categoryId"]) => {
  if (categoryId === "uncategorized") return ticket.category_id === "";
  if (categoryId) return ticket.category_id === categoryId;
  return true;
};

const ticketMatchesFilters = (ticket: Ticket, filters: BoardFilters, sets: FilterSets) =>
  (filters.projectIds.length === 0 || sets.projectIds.has(ticket.project_id)) &&
  matchesCategory(ticket, filters.categoryId) &&
  (!filters.typeId || ticket.type_id === filters.typeId) &&
  (filters.statusIds.length === 0 || sets.statusIds.has(ticket.status)) &&
  (filters.labels.length === 0 || (ticket.labels ?? []).some((l) => sets.labels.has(l))) &&
  (filters.developers.length === 0 || sets.developers.has(ticket.developer)) &&
  (!filters.waitingForMeToTest || (ticket.tester === sets.myLogin && sets.testingStatusIds.has(ticket.status)));

interface FilterSets {
  projectIds: Set<string>;
  labels: Set<string>;
  statusIds: Set<string>;
  developers: Set<string>;
  testingStatusIds: Set<string>;
  myLogin: string;
}

// projectId, when given, locks the board to that project, seeded into initial state so the URL is the source of truth.
export const useBoardFilters = (tickets: Ticket[] | undefined, projectId?: string) => {
  const [filters, setFilters] = useState<BoardFilters>(() =>
    projectId ? { ...EMPTY_FILTERS, projectIds: [projectId] } : EMPTY_FILTERS,
  );

  // The lazy useState above only fires once, before an in-place redirect resolves the project; re-seed here too.
  useEffect(() => {
    if (projectId) setFilters((current) => ({ ...current, projectIds: [projectId] }));
  }, [projectId]);

  const { data: me } = useFetchMe();
  const { data: statuses } = useFetchProjectStatuses(projectId);
  const myLogin = me?.user.login ?? "";

  const developers = useMemo(() => {
    const set = new Set((tickets ?? []).map((t) => t.developer).filter((d) => d !== ""));
    return Array.from(set).sort();
  }, [tickets]);

  // Offered once the signed-in member tests anything here, and kept while on so it can be switched off.
  const showWaitingForMeToTest =
    filters.waitingForMeToTest || (myLogin !== "" && (tickets ?? []).some((t) => t.tester === myLogin));

  const filteredTickets = useMemo(() => {
    const sets: FilterSets = {
      projectIds: new Set(filters.projectIds),
      labels: new Set(filters.labels),
      statusIds: new Set(filters.statusIds),
      developers: new Set(filters.developers),
      testingStatusIds: new Set((statuses ?? []).filter((s) => s.kind === StatusKind.Testing).map((s) => s.id)),
      myLogin,
    };
    return (tickets ?? []).filter((t) => ticketMatchesFilters(t, filters, sets));
  }, [tickets, filters, statuses, myLogin]);

  const toggleProject = (projectId: string) =>
    setFilters((current) => ({
      ...current,
      projectIds: current.projectIds.includes(projectId)
        ? current.projectIds.filter((id) => id !== projectId)
        : [...current.projectIds, projectId],
    }));

  const toggleCategory = (categoryId: string | null) =>
    setFilters((current) => ({
      ...current,
      categoryId: current.categoryId === categoryId ? null : categoryId,
    }));

  const toggleLabel = (label: string) =>
    setFilters((current) => ({
      ...current,
      labels: current.labels.includes(label)
        ? current.labels.filter((l) => l !== label)
        : [...current.labels, label],
    }));

  const selectType = (typeId: string | null) =>
    setFilters((current) => ({ ...current, typeId: current.typeId === typeId ? null : typeId }));

  const toggleStatus = (statusId: string) =>
    setFilters((current) => ({
      ...current,
      statusIds: current.statusIds.includes(statusId)
        ? current.statusIds.filter((id) => id !== statusId)
        : [...current.statusIds, statusId],
    }));

  const toggleDeveloper = (developer: string) =>
    setFilters((current) => ({
      ...current,
      developers: current.developers.includes(developer)
        ? current.developers.filter((d) => d !== developer)
        : [...current.developers, developer],
    }));

  const toggleWaitingForMeToTest = () =>
    setFilters((current) => ({ ...current, waitingForMeToTest: !current.waitingForMeToTest }));

  // Resets what the user picked, never the URL's project scope, or the view would leak every project's tickets.
  const clearAll = () =>
    setFilters(projectId ? { ...EMPTY_FILTERS, projectIds: [projectId] } : EMPTY_FILTERS);

  return {
    filters,
    developers,
    showWaitingForMeToTest,
    filteredTickets,
    toggleProject,
    toggleCategory,
    toggleLabel,
    selectType,
    toggleStatus,
    toggleDeveloper,
    toggleWaitingForMeToTest,
    clearAll,
  };
};
