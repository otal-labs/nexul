import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayButton } from "@/components/play/PlayButton";
import type { Play } from "@/models/Play";
import type { Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play: Play = {
  id: "play-1",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "Implement the ticket and open a PR",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  created_by: "u-1",
  created_at: "",
  updated_at: "",
};

const runningTrail = (starterId: string): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c-1",
  starter_id: starterId,
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  move_to_status_id: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "running",
  started_at: "2026-09-17T10:00:00Z",
  ended_at: null,
  last_error: "",
  reply_message_id: "",
  activity: [],
});

interface Scenario {
  permissions?: string[];
  resolve?: unknown;
  presence?: Record<string, string>;
  trails?: Trail[];
}

const mockApi = ({ permissions = ["plays:run"], resolve = { ok: true, computer_id: "c-1" }, presence = { "c-1": "connected" }, trails = [] }: Scenario) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/auth/me") return { data: { user: { id: "u-me", login: "me" } } };
    if (url === "/api/pairing/resolve") return { data: resolve };
    if (url === "/api/pairing/presence") return { data: { computers: presence } };
    if (url === "/api/pairing/computers") return { data: { computers: [{ id: "c-1", name: "Onik's PC" }] } };
    if (url === "/api/pairing/computers/c-1/providers") return { data: { providers: [] } };
    if (url === "/api/plays/runs") return { data: trails };
    if (url === "/api/plays/latest-choices")
      return { data: { memory_ids: [], move_to_status_id: "", computer_id: "", provider: "", model: "" } };
    return { data: [] };
  });

const renderButton = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <PlayButton play={play} projectId="p-1" targetType="ticket" targetId="t-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("PlayButton", () => {
  it("is hidden without plays:run", async () => {
    mockApi({ permissions: ["plays:read"] });
    renderButton();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("is disabled with the readiness reason when no harness is paired", async () => {
    mockApi({ resolve: { ok: false, reason: "unpaired" } });
    renderButton();
    expect(await screen.findByText("Pair a harness in Settings to run plays")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fix with AI" })).toBeDisabled();
  });

  it("is disabled with the offline reason when the resolved harness is not connected", async () => {
    mockApi({ presence: {} });
    renderButton();
    expect(await screen.findByText("Your harness is offline")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fix with AI" })).toBeDisabled();
  });

  it("is disabled with 'a run is in progress' when someone else's trail is active", async () => {
    mockApi({ trails: [runningTrail("u-other")] });
    renderButton();
    expect(await screen.findByText("a run is in progress")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fix with AI" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Stop Fix with AI" })).not.toBeInTheDocument();
  });

  it("shows the running button with Stop to the starter, and Stop posts the stop", async () => {
    const user = userEvent.setup();
    mockApi({ trails: [runningTrail("u-me")] });
    vi.mocked(api.post).mockResolvedValue({ data: { ...runningTrail("u-me"), state: "interrupted" } });
    renderButton();

    await user.click(await screen.findByRole("button", { name: "Stop Fix with AI" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/runs/tr-1/stop");
  });

  it("reads 'Waiting for your answer' when the run stopped on a question, and opens the trail with the card", async () => {
    const user = userEvent.setup();
    const waiting: Trail = {
      ...runningTrail("u-me"),
      state: "waiting",
      question: { request_id: "req-1", questions: [{ id: "q1", text: "Proceed?", options: [{ label: "Yes" }] }], asked_at: "2026-09-17T10:01:00Z" },
    };
    vi.mocked(api.get).mockImplementation(async (url: string) => {
      if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["plays:run"] } };
      if (url === "/api/auth/me") return { data: { user: { id: "u-me", login: "me" } } };
      if (url === "/api/pairing/resolve") return { data: { ok: true, computer_id: "c-1" } };
      if (url === "/api/pairing/presence") return { data: { computers: { "c-1": "connected" } } };
      if (url === "/api/plays/runs") return { data: [waiting] };
      if (url === "/api/plays/runs/tr-1") return { data: waiting };
      return { data: [] };
    });
    renderButton();

    const button = await screen.findByRole("button", { name: "Waiting for your answer" });
    expect(screen.getByRole("button", { name: "Stop Fix with AI" })).toBeInTheDocument();
    await user.click(button);
    expect(await screen.findByRole("heading", { name: "Proceed?" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send answer" })).toBeInTheDocument();
  });

  it("a sibling play stays disabled with the waiting reason while another play's run waits", async () => {
    mockApi({ trails: [{ ...runningTrail("u-other"), play_id: "play-other", state: "waiting" }] });
    renderButton();
    expect(await screen.findByText("a run is waiting for an answer")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Fix with AI" })).toBeDisabled();
  });

  it("shows Stop to a plays:write holder who did not start the run", async () => {
    mockApi({ permissions: ["plays:run", "plays:write"], trails: [runningTrail("u-other")] });
    renderButton();
    expect(await screen.findByRole("button", { name: "Stop Fix with AI" })).toBeInTheDocument();
  });

  it("is enabled when the harness is ready and nothing runs, and opens the run dialog", async () => {
    const user = userEvent.setup();
    mockApi({});
    renderButton();
    const button = await screen.findByRole("button", { name: "Fix with AI" });
    await waitFor(() => expect(button).toBeEnabled());
    expect(screen.queryByText("a run is in progress")).not.toBeInTheDocument();

    await user.click(button);
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Fix with AI" })).toBeInTheDocument();
  });

  it("flips to running from a live frame without a refetch of readiness", async () => {
    mockApi({ trails: [{ ...runningTrail("u-other"), state: "done", ended_at: "2026-09-17T10:05:00Z" }] });
    renderButton();
    const button = await screen.findByRole("button", { name: "Fix with AI" });
    await waitFor(() => expect(button).toBeEnabled());

    usePlayRunStore.getState().applyFrame({
      trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", state: "running", activity: null, ended_at: null, last_error: "",
    });
    expect(await screen.findByText("a run is in progress")).toBeInTheDocument();
  });
});
