import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { useBoardFilters } from "@/hooks/useBoardFilters";
import type { Ticket, TicketStatus } from "@/models/Ticket";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn() },
  errorMessage: vi.fn(),
}));

const ticket = (id: string, projectId: string, overrides: Partial<Ticket> = {}): Ticket => ({
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
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "",
  updated_at: "",
  labels: [],
  ...overrides,
});

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    {children}
  </QueryClientProvider>
);

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/auth/me") return { data: { user: { login: "lena" } } };
    if (url === "/api/statuses") {
      return {
        data: [
          { id: "open", name: "Open", kind: "backlog", icon: "Circle", position: 0 },
          { id: "qa", name: "QA", kind: "testing", icon: "Circle", position: 1 },
        ],
      };
    }
    return { data: [] };
  });
});

describe("useBoardFilters", () => {
  it("shows all tickets when no project is locked", () => {
    const tickets = [ticket("t-1", "p-1"), ticket("t-2", "p-2")];
    const { result } = renderHook(() => useBoardFilters(tickets), { wrapper });
    expect(result.current.filteredTickets).toHaveLength(2);
    expect(result.current.filters.projectIds).toEqual([]);
  });

  it("scopes to a single project when a projectId is locked", () => {
    const tickets = [ticket("t-1", "p-1"), ticket("t-2", "p-2")];
    const { result } = renderHook(() => useBoardFilters(tickets, "p-1"), { wrapper });
    expect(result.current.filteredTickets.map((t) => t.id)).toEqual(["t-1"]);
    expect(result.current.filters.projectIds).toEqual(["p-1"]);
  });

  it("narrows to the picked developers", () => {
    const tickets = [ticket("t-1", "p-1", { developer: "onik97" }), ticket("t-2", "p-1", { developer: "bob" })];
    const { result } = renderHook(() => useBoardFilters(tickets, "p-1"), { wrapper });
    expect(result.current.developers).toEqual(["bob", "onik97"]);
    act(() => result.current.toggleDeveloper("bob"));
    expect(result.current.filteredTickets.map((t) => t.id)).toEqual(["t-2"]);
  });

  it("waiting for me to test keeps only my tickets in testing-stage columns", async () => {
    const tickets = [
      ticket("mine-testing", "p-1", { tester: "lena", status: "qa" as TicketStatus }),
      ticket("mine-backlog", "p-1", { tester: "lena", status: "open" as TicketStatus }),
      ticket("theirs-testing", "p-1", { tester: "bob", status: "qa" as TicketStatus }),
    ];
    const { result } = renderHook(() => useBoardFilters(tickets, "p-1"), { wrapper });
    await waitFor(() => expect(result.current.showWaitingForMeToTest).toBe(true));

    act(() => result.current.toggleWaitingForMeToTest());
    await waitFor(() => expect(result.current.filteredTickets.map((t) => t.id)).toEqual(["mine-testing"]));

    act(() => result.current.clearAll());
    expect(result.current.filters.waitingForMeToTest).toBe(false);
    expect(result.current.filteredTickets).toHaveLength(3);
  });

  it("does not offer the toggle when I test nothing on this board", async () => {
    const tickets = [ticket("t-1", "p-1", { tester: "bob", status: "qa" as TicketStatus })];
    const { result } = renderHook(() => useBoardFilters(tickets, "p-1"), { wrapper });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/auth/me"));
    expect(result.current.showWaitingForMeToTest).toBe(false);
  });
});
