import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useBoardFilters } from "@/hooks/useBoardFilters";
import type { Ticket } from "@/models/Ticket";

const ticket = (id: string, projectId: string): Ticket => ({
  id,
  project_id: projectId,
  category_id: "",
  type_id: "ticket-type-task",
  title: "t",
  body: "",
  status: "open",
  position: 0,
  number: 1,
  doc_id: "",
  assignee: "",
  created_at: "",
  updated_at: "",
  labels: [],
});

describe("useBoardFilters", () => {
  it("shows all tickets when no project is locked", () => {
    const tickets = [ticket("t-1", "p-1"), ticket("t-2", "p-2")];
    const { result } = renderHook(() => useBoardFilters(tickets));
    expect(result.current.filteredTickets).toHaveLength(2);
    expect(result.current.filters.projectIds).toEqual([]);
  });

  it("scopes to a single project when a projectId is locked", () => {
    const tickets = [ticket("t-1", "p-1"), ticket("t-2", "p-2")];
    const { result } = renderHook(() => useBoardFilters(tickets, "p-1"));
    expect(result.current.filteredTickets.map((t) => t.id)).toEqual(["t-1"]);
    expect(result.current.filters.projectIds).toEqual(["p-1"]);
  });
});
