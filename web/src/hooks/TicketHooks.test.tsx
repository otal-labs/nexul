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
} from "@/hooks/TicketHooks";
import { TicketStatus } from "@/models/Ticket";
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
  assignee: "onik97",
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
    await result.current.mutateAsync({ title: "Fix storage", body: "write migrations", project_id: "p-1", doc_id: "doc-1", assignee: "onik97" });
    expect(api.post).toHaveBeenCalledWith("/api/tickets", {
      title: "Fix storage",
      body: "write migrations",
      project_id: "p-1",
      doc_id: "doc-1",
      assignee: "onik97",
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

  it("invalidates the board, single-ticket, and scoped-list keys", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, status: TicketStatus.Done } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useUpdateTicketStatus(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "t-1", status: TicketStatus.Done });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicket", "t-1"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicketsByDoc"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicketsByProject"] });
  });
});

describe("useUpdateTicket", () => {
  it("patches the title and body", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, title: "New title", body: "New body" } });
    const { result } = renderHook(() => useUpdateTicket(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", title: "New title", body: "New body" });
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1", { title: "New title", body: "New body" });
  });

  it("invalidates the board, single-ticket, and scoped-list keys", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, title: "New title" } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useUpdateTicket(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "t-1", title: "New title", body: "" });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicket", "t-1"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicketsByDoc"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicketsByProject"] });
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

describe("useUpdateTicketPosition", () => {
  it("patches the position", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, position: 3 } });
    const { result } = renderHook(() => useUpdateTicketPosition(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", position: 3 });
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/position", { position: 3 });
  });

  it("invalidates the board and single-ticket keys, without a success toast", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, position: 3 } });
    vi.mocked(toast.success).mockClear();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const invalidateSpy = vi.spyOn(client, "invalidateQueries");
    const { result } = renderHook(() => useUpdateTicketPosition(), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    await result.current.mutateAsync({ id: "t-1", position: 3 });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTickets"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["getTicket", "t-1"] });
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.patch).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useUpdateTicketPosition(), { wrapper });
    await result.current.mutateAsync({ id: "t-1", position: 3 }).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
