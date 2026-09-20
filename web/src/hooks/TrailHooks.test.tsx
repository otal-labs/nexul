import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import {
  useActiveTrail,
  useFetchActiveTrails,
  useFetchLatestChoices,
  useFetchTrails,
  useIsTicketRunActive,
  useRunPlay,
  useStopTrail,
} from "@/hooks/TrailHooks";
import type { Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const trail = (overrides: Partial<Trail> = {}): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: "u-1",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  move_to_status_id: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "done",
  started_at: "2026-09-17T10:00:00Z",
  ended_at: "2026-09-17T10:05:00Z",
  last_error: "",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

let client: QueryClient;
const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("useFetchTrails / useFetchLatestChoices", () => {
  it("lists a target's trails", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [trail()] });
    const { result } = renderHook(() => useFetchTrails("ticket", "t-1"), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.get).toHaveBeenCalledWith("/api/plays/runs", { params: { target_type: "ticket", target_id: "t-1" } });
  });

  it("reads the caller's latest choices for a play in a project", async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: { memory_ids: ["m-1"], move_to_status_id: "st-1", computer_id: "c-1", provider: "claude", model: "sonnet-5" },
    });
    const { result } = renderHook(() => useFetchLatestChoices("play-1", "p-1"), { wrapper });
    await waitFor(() =>
      expect(result.current.data).toEqual({
        memory_ids: ["m-1"],
        move_to_status_id: "st-1",
        computer_id: "c-1",
        provider: "claude",
        model: "sonnet-5",
      }),
    );
    expect(api.get).toHaveBeenCalledWith("/api/plays/latest-choices", { params: { play_id: "play-1", project_id: "p-1" } });
  });
});

describe("useRunPlay", () => {
  it("posts the run and puts the returned starting trail at the top of the target's list", async () => {
    const started = trail({ id: "tr-2", state: "starting", ended_at: null });
    client.setQueryData(["getTrails", "ticket", "t-1"], [trail()]);
    vi.mocked(api.get).mockResolvedValue({ data: [trail(), started] });
    vi.mocked(api.post).mockResolvedValue({ data: started });
    const { result } = renderHook(() => useRunPlay(), { wrapper });

    await result.current.mutateAsync({
      playId: "play-1",
      input: {
        target_type: "ticket",
        target_id: "t-1",
        memory_ids: ["m-1"],
        custom_instructions: "go",
        move_to_status_id: "st-1",
        computer_id: "c-1",
        provider: "claude",
        model: "sonnet-5",
      },
    });

    expect(api.post).toHaveBeenCalledWith("/api/plays/play-1/run", {
      target_type: "ticket",
      target_id: "t-1",
      memory_ids: ["m-1"],
      custom_instructions: "go",
      move_to_status_id: "st-1",
      computer_id: "c-1",
      provider: "claude",
      model: "sonnet-5",
    });
    expect((client.getQueryData(["getTrails", "ticket", "t-1"]) as Trail[])[0]?.id).toBe("tr-2");
  });

  it("does not toast an api error, leaving it to the dialog", async () => {
    const { toast } = await import("sonner");
    vi.mocked(api.post).mockRejectedValue(new Error("over the ceiling"));
    const { result } = renderHook(() => useRunPlay(), { wrapper });
    await result.current
      .mutateAsync({
        playId: "play-1",
        input: {
          target_type: "ticket",
          target_id: "t-1",
          memory_ids: [],
          custom_instructions: "",
          move_to_status_id: "",
          computer_id: "",
          provider: "",
          model: "",
        },
      })
      .catch(() => {});
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(toast.error).not.toHaveBeenCalled();
  });
});

describe("useStopTrail", () => {
  it("posts the stop and patches the interrupted trail into the cached list", async () => {
    const running = trail({ state: "running", ended_at: null });
    client.setQueryData(["getTrails", "ticket", "t-1"], [running]);
    vi.mocked(api.get).mockResolvedValue({ data: [] });
    vi.mocked(api.post).mockResolvedValue({ data: { ...running, state: "interrupted", last_error: "stopped by onik" } });
    const { result } = renderHook(() => useStopTrail(), { wrapper });

    await result.current.mutateAsync("tr-1");

    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/stop");
    expect((client.getQueryData(["getTrails", "ticket", "t-1"]) as Trail[])[0]?.state).toBe("interrupted");
  });
});

describe("useActiveTrail", () => {
  it("returns the fetched active trail, and drops it once a live frame ends it", async () => {
    vi.mocked(api.get).mockResolvedValue({ data: [trail({ state: "running", ended_at: null }), trail({ id: "tr-0" })] });
    const { result } = renderHook(() => useActiveTrail("ticket", "t-1"), { wrapper });
    await waitFor(() => expect(result.current?.id).toBe("tr-1"));

    usePlayRunStore.getState().applyFrame({
      trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", state: "done", activity: null, ended_at: null, last_error: "",
    });
    await waitFor(() => expect(result.current).toBeUndefined());
  });
});

describe("useFetchActiveTrails / useIsTicketRunActive", () => {
  const mockBoard = (active: Record<string, string>) =>
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/tickets") return { data: [{ id: "t-1", project_id: "p-1" }, { id: "t-2", project_id: "p-1" }] };
      if (url === "/api/plays/runs/active") return { data: { active } };
      return { data: [] };
    });

  it("asks once per project for every ticket id on the board", async () => {
    mockBoard({ "t-1": "tr-1" });
    const { result } = renderHook(() => useFetchActiveTrails("p-1"), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual({ "t-1": "tr-1" }));
    expect(api.get).toHaveBeenCalledWith("/api/plays/runs/active", { params: { ticket_ids: "t-1,t-2" } });
  });

  it("reads the on-load answer, then follows live frames without a refetch", async () => {
    mockBoard({ "t-1": "tr-1" });
    const { result } = renderHook(() => ({ t1: useIsTicketRunActive("p-1", "t-1"), t2: useIsTicketRunActive("p-1", "t-2") }), { wrapper });
    await waitFor(() => expect(result.current.t1).toBe(true));
    expect(result.current.t2).toBe(false);

    const base = { play_id: "play-1", target_type: "ticket" as const, activity: null, ended_at: null, last_error: "" };
    usePlayRunStore.getState().applyFrame({ ...base, trail_id: "tr-1", target_id: "t-1", state: "done" });
    usePlayRunStore.getState().applyFrame({ ...base, trail_id: "tr-2", target_id: "t-2", state: "starting" });
    await waitFor(() => expect(result.current.t1).toBe(false));
    expect(result.current.t2).toBe(true);
  });
});
