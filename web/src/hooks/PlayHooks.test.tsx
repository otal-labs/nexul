import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useApplicableTicketPlays,
  useCreatePlay,
  useDeletePlay,
  useFetchApplicablePlays,
  useFetchWorkspacePlays,
  useUpdatePlay,
} from "@/hooks/PlayHooks";
import type { SavePlayFormData } from "@/models/Play";
import type { Ticket, TicketStatus } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const ticketAt = (status: string): Ticket => ({
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "",
  title: "Fix login",
  body: "",
  status: status as TicketStatus,
  position: 0,
  number: 1,
  doc_id: "",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "",
  updated_at: "",
});

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const wrapper = ({ children }: { children: ReactNode }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
};

const savePlayInput = (overrides: Partial<SavePlayFormData> = {}): SavePlayFormData => ({
  label: "Fix with AI",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  ...overrides,
});

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.delete).mockReset();
});

describe("useFetchWorkspacePlays", () => {
  it("fetches the workspace's plays", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ id: "play-1", label: "Fix with AI" }] });
    const { result } = renderHook(() => useFetchWorkspacePlays("ws-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays");
    expect(result.current.data).toEqual([{ id: "play-1", label: "Fix with AI" }]);
  });

  it("does not fetch with an empty workspace id", () => {
    renderHook(() => useFetchWorkspacePlays(""), { wrapper });
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useCreatePlay", () => {
  it("posts the wire shape, converting an empty stage to null", async () => {
    vi.mocked(api.post).mockResolvedValue({ data: { id: "play-1" } });
    const { result } = renderHook(() => useCreatePlay("ws-1"), { wrapper });
    await result.current.mutateAsync(savePlayInput({ type: "doc", show_when_stage: "" }));
    expect(api.post).toHaveBeenCalledWith("/api/workspaces/ws-1/plays", {
      label: "Fix with AI",
      type: "doc",
      description: "",
      instructions: "",
      enabled: true,
      show_when_stage: null,
      excluded_project_ids: [],
    });
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useCreatePlay("ws-1"), { wrapper });
    await result.current.mutateAsync(savePlayInput()).catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useFetchApplicablePlays", () => {
  it("asks for the plays that apply to a project at a stage", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [{ id: "play-1", label: "Fix with AI" }] });
    const { result } = renderHook(() => useFetchApplicablePlays("ws-1", "p-1", "ticket", "progress"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", {
      params: { project_id: "p-1", type: "ticket", stage: "progress" },
    });
  });

  it("waits for the ticket's stage before asking", () => {
    renderHook(() => useFetchApplicablePlays("ws-1", "p-1", "ticket", undefined), { wrapper });
    expect(api.get).not.toHaveBeenCalled();
  });
});

describe("useApplicableTicketPlays", () => {
  it("resolves the ticket's column to its stage kind and asks with it", async () => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/statuses") return { data: [{ id: "st-progress", name: "In progress", kind: "progress", position: 1 }] };
      if (url === "/api/workspaces/ws-1/plays/applicable") return { data: [{ id: "play-1", label: "Fix with AI" }] };
      return { data: [] };
    });
    const { result } = renderHook(() => useApplicableTicketPlays(ticketAt("st-progress")), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual([{ id: "play-1", label: "Fix with AI" }]));
    expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", {
      params: { project_id: "p-1", type: "ticket", stage: "progress" },
    });
  });
});

describe("useUpdatePlay", () => {
  it("patches the given play", async () => {
    vi.mocked(api.patch).mockResolvedValue({ data: { id: "play-1" } });
    const { result } = renderHook(() => useUpdatePlay("ws-1"), { wrapper });
    await result.current.mutateAsync({ playId: "play-1", input: savePlayInput({ label: "Renamed" }) });
    expect(api.patch).toHaveBeenCalledWith(
      "/api/workspaces/ws-1/plays/play-1",
      expect.objectContaining({ label: "Renamed" }),
    );
  });
});

describe("useDeletePlay", () => {
  it("deletes the given play", async () => {
    vi.mocked(api.delete).mockResolvedValue({ data: undefined });
    const { result } = renderHook(() => useDeletePlay("ws-1"), { wrapper });
    await result.current.mutateAsync("play-1");
    expect(api.delete).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/play-1");
  });

  it("surfaces an api error", async () => {
    vi.mocked(api.delete).mockRejectedValue(new Error("boom"));
    const { result } = renderHook(() => useDeletePlay("ws-1"), { wrapper });
    await result.current.mutateAsync("play-1").catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
