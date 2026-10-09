import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api, errorMessage } from "@/api/client";
import {
  useCreateTicket,
  useFetchTicket,
  useFetchTickets,
  useFetchTicketsByDoc,
  useUpdateTicket,
  useUpdateTicketPosition,
  useUpdateTicketStatus,
  ticketFollower,
} from "@/hooks/TicketHooks";
import { TicketStatus, type Ticket } from "@/models/Ticket";
import { followFrame, isStale, seeded } from "@/test/followFrame";
vi.mock("@/api/client", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket = {
  id: "t-1",
  project_id: "p-1",
  title: "Fix storage",
  body: "write migrations",
  status: TicketStatus.Open,
  doc_id: "doc-1",
  developer: "onik97",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useFetchTickets", () => {
  it("loads the ticket list", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [ticket] });
    const { result } = renderHook(() => useFetchTickets(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([ticket]));
    expect(api.get).toHaveBeenCalledWith("/api/tickets");
  });
});

describe("useFetchTicket", () => {
  it("loads a single ticket", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: ticket });
    const { result } = renderHook(() => useFetchTicket("t-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(ticket));
    expect(api.get).toHaveBeenCalledWith("/api/tickets/t-1");
  });
});

describe("useFetchTicketsByDoc", () => {
  it("scopes by doc", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [ticket] });
    const { result } = renderHook(() => useFetchTicketsByDoc("doc-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([ticket]));
    expect(api.get).toHaveBeenCalledWith("/api/tickets", { params: { doc_id: "doc-1" } });
  });
});

describe("useCreateTicket", () => {
  it("posts and invalidates", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: ticket });
    const { result } = renderHook(() => useCreateTicket(), { wrapper });
    await result.current.mutateAsync({ title: "Fix storage", body: "write migrations", project_id: "p-1", doc_id: "doc-1", developer: "onik97" });
    expect(api.post).toHaveBeenCalledWith("/api/tickets", {
      title: "Fix storage",
      body: "write migrations",
      project_id: "p-1",
      doc_id: "doc-1",
      developer: "onik97",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreateTicket(), { wrapper });
    await result.current.mutateAsync({ title: "x", body: "", project_id: "p-1" }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useUpdateTicketStatus", () => {
  it("patches the status", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, status: TicketStatus.Done } });
    const { result } = renderHook(() => useUpdateTicketStatus(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", status: TicketStatus.Done });
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "done" });
  });

});

describe("useUpdateTicket", () => {
  it("patches the title and body", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, title: "New title", body: "New body" } });
    const { result } = renderHook(() => useUpdateTicket(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", title: "New title", body: "New body" });
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1", { title: "New title", body: "New body" });
  });


  it("surfaces the error toast on failure", async () => {
    vi.mocked(api.patch).mockRejectedValue(new Error("boom"));
    vi.mocked(errorMessage).mockReturnValue("boom");
    const { result } = renderHook(() => useUpdateTicket(), { wrapper });
    await expect(
      result.current.mutateAsync({ id: "t-1", title: "New title", body: "" }),
    ).rejects.toThrow();
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("boom"));
  });
});

// Each edit hands the server's ticket to the cache, so every view shows it without a refetch.
type Wrapper = ({ children }: { children: ReactNode }) => ReactNode;
describe.each([
  ["useUpdateTicket", async (w: Wrapper) => renderHook(() => useUpdateTicket(), { wrapper: w }).result.current.mutateAsync({ id: "t-1", title: "New title", body: "" })],
  ["useUpdateTicketStatus", async (w: Wrapper) => renderHook(() => useUpdateTicketStatus(), { wrapper: w }).result.current.mutateAsync({ id: "t-1", status: TicketStatus.Done })],
])("%s", (_, save) => {
  it("puts the server's ticket in every cached view", async () => {
    const saved = { ...ticket, title: "New title", status: TicketStatus.Done, updated_at: "2026-08-02T12:00:05Z" };
    vi.mocked(api.patch).mockResolvedValue({ data: saved });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const views = [["getTickets"], ["getTicketsByDoc", "doc-1"], ["getTicketsByProject", "p-1"]];
    for (const key of views) client.setQueryData(key, [ticket]);
    client.setQueryData(["getTicket", "t-1"], ticket);
    await save(({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>);
    for (const key of views) expect(client.getQueryData(key)).toEqual([saved]);
    expect(client.getQueryData(["getTicket", "t-1"])).toEqual(saved);
  });
});

describe("useUpdateTicketPosition", () => {
  it("patches the position", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, position: 3 } });
    const { result } = renderHook(() => useUpdateTicketPosition(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", position: 3 });
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/position", { position: 3 });
  });

  it("takes the server's new position into the board without a success toast", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, position: 3 } });
    vi.mocked(toast.success).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(["getTickets"], [ticket]);
    const { result } = renderHook(() => useUpdateTicketPosition(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "t-1", position: 3 });
    expect(client.getQueryData<{ position: number }[]>(["getTickets"])?.[0]?.position).toBe(3);
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.patch).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useUpdateTicketPosition(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", position: 3 }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("the ticket follower", () => {
  const inColumn = (id: string, status: string) => ({ id, project_id: "p-1", doc_id: "", status, title: id, updated_at: "" });

  it("refetches the tickets of a deleted column and the lists holding them, and leaves other lists alone", async () => {
    const client = seeded([
      [["getTicketsByProject", "p-1"], [inColumn("t-1", "st-gone"), inColumn("t-2", "st-kept")]],
      [["getTicketsByProject", "p-2"], [{ ...inColumn("t-3", "st-other"), project_id: "p-2" }]],
      [["getTicket", "t-1"], inColumn("t-1", "st-gone")],
      [["getTicket", "t-2"], inColumn("t-2", "st-kept")],
    ]);
    await followFrame(ticketFollower, "status.deleted", { status: { id: "st-gone", project_id: "p-1" } }, client);
    expect([["getTicketsByProject", "p-1"], ["getTicket", "t-1"]].map((key) => isStale(client, key))).toEqual([true, true]);
    expect([["getTicketsByProject", "p-2"], ["getTicket", "t-2"]].map((key) => isStale(client, key))).toEqual([false, false]);
  });

  it("moves a ticket to the end of its column in the new category without a request", async () => {
    const card = (id: string, category_id: string, position: number) => ({ ...inColumn(id, "st-1"), category_id, position });
    const client = seeded([
      [["getTicketsByProject", "p-1"], [card("t-1", "cat-a", 0), card("t-2", "cat-b", 0), card("t-3", "cat-b", 4), { ...card("t-4", "cat-b", 9), status: "st-2" }]],
      [["getTicket", "t-1"], card("t-1", "cat-a", 0)],
    ]);
    await followFrame(ticketFollower, "ticket.category_changed", { ticket_id: "t-1", category_id: "cat-b", project_id: "p-1" }, client);
    const moved = client.getQueryData<Ticket[]>(["getTicketsByProject", "p-1"])?.find((t) => t.id === "t-1");
    expect([moved?.category_id, moved?.position]).toEqual(["cat-b", 5]);
    expect(client.getQueryData<Ticket>(["getTicket", "t-1"])?.category_id).toBe("cat-b");
    expect([["getTicketsByProject", "p-1"], ["getTicket", "t-1"]].map((key) => isStale(client, key))).toEqual([false, false]);
  });

  it("refetches the lists holding a moved ticket when its project's list is not loaded", async () => {
    const client = seeded([[["getTickets"], [inColumn("t-1", "st-1")]]]);
    await followFrame(ticketFollower, "ticket.category_changed", { ticket_id: "t-1", category_id: "cat-b", project_id: "p-1" }, client);
    expect(isStale(client, ["getTickets"])).toBe(true);
  });

  it("refetches a ticket and its branches once its run ends, and nothing for a run still going or on a doc", async () => {
    const client = seeded([
      [["getTickets"], [inColumn("t-9", "st-1")]],
      [["getTicket", "t-9"], inColumn("t-9", "st-1")],
      [["getTicketLinks", "t-9"], { prs: [], branches: [] }],
    ]);
    const run = { trail_id: "tr-1", play_id: "pl-1", target_type: "ticket", target_id: "t-9", activity: null, ended_at: null, last_error: "" };
    await followFrame(ticketFollower, "play.run", { ...run, state: "running" }, client);
    await followFrame(ticketFollower, "play.run", { ...run, target_type: "doc", state: "done" }, client);
    expect(isStale(client, ["getTicket", "t-9"])).toBe(false);

    await followFrame(ticketFollower, "play.run", { ...run, state: "done" }, client);
    expect([["getTickets"], ["getTicket", "t-9"], ["getTicketLinks", "t-9"]].map((key) => isStale(client, key))).toEqual([true, true, true]);
  });
});
