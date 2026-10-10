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
  playFollower,
} from "@/hooks/PlayHooks";
import type { SavePlayFormData } from "@/models/Play";
import type { Ticket, TicketStatus } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { followFrame, isStale, seeded } from "@/test/followFrame";

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

describe("the play follower", () => {
  it("refetches the plays and run menus of the play's workspace only", async () => {
    const client = seeded([
      [["getWorkspacePlays", "ws-1"], []],
      [["getWorkspacePlays", "ws-2"], []],
      [["getApplicablePlays", "ws-1", "p-1", "ticket", null], []],
      [["getApplicablePlays", "ws-2", "p-2", "ticket", null], []],
    ]);
    await followFrame(playFollower, "play.updated", { play: { id: "pl-1", workspace_id: "ws-2" } }, client);
    const keys = [["getWorkspacePlays", "ws-1"], ["getWorkspacePlays", "ws-2"], ["getApplicablePlays", "ws-1", "p-1", "ticket", null], ["getApplicablePlays", "ws-2", "p-2", "ticket", null]];
    expect(keys.map((key) => isStale(client, key))).toEqual([false, true, false, true]);

    await followFrame(playFollower, "play.deleted", { id: "pl-1", label: "Review", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getWorkspacePlays", "ws-1"])).toBe(true);
  });

  it("refetches only the auto plays of the play a frame names", async () => {
    const client = seeded([
      [["getAutoPlays", "ws-1", "pl-1"], []],
      [["getAutoPlays", "ws-1", "pl-2"], []],
    ]);
    await followFrame(playFollower, "auto_play.updated", { auto_play: { id: "ap-1", play_id: "pl-1", workspace_id: "ws-1" } }, client);
    expect([isStale(client, ["getAutoPlays", "ws-1", "pl-1"]), isStale(client, ["getAutoPlays", "ws-1", "pl-2"])]).toEqual([true, false]);

    await followFrame(playFollower, "auto_play.deleted", { id: "ap-2", play_id: "pl-2", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getAutoPlays", "ws-1", "pl-2"])).toBe(true);
  });

  it("patches the daily cap of the workspace a frame names, with no request", async () => {
    const client = seeded([
      [["getAutoPlayLimits", "ws-1"], { daily_cap_per_ticket: 5 }],
      [["getAutoPlayLimits", "ws-2"], { daily_cap_per_ticket: 5 }],
    ]);
    await followFrame(playFollower, "auto_play.limits_updated", { workspace_id: "ws-1", daily_cap_per_ticket: 9 }, client);
    expect(client.getQueryData(["getAutoPlayLimits", "ws-1"])).toEqual({ daily_cap_per_ticket: 9 });
    expect(client.getQueryData(["getAutoPlayLimits", "ws-2"])).toEqual({ daily_cap_per_ticket: 5 });
    expect(isStale(client, ["getAutoPlayLimits", "ws-1"])).toBe(false);
  });

  it("drops a deleted play's auto plays with it", async () => {
    const client = seeded([[["getAutoPlays", "ws-1", "pl-1"], []]]);
    await followFrame(playFollower, "play.deleted", { id: "pl-1", label: "Review", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getAutoPlays", "ws-1", "pl-1"])).toBe(true);
  });
});
