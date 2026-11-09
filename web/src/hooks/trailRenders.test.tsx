import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { act, Profiler } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Conversation } from "@nexul/client-core/chat";
import { unknownPerson } from "@nexul/client-core/person";

import { api } from "@/api/client";
import { MessageList } from "@/components/chat/MessageList";
import { PlayButton } from "@/components/play/PlayButton";
import type { ListedPlay } from "@/models/Play";
import type { RunFrame, Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const play: ListedPlay = {
  id: "play-1",
  workspace_id: "ws-1",
  label: "Fix with AI",
  type: "ticket",
  description: "",
  instructions: "",
  enabled: true,
  show_when_stage: "progress",
  excluded_project_ids: [],
  builtin_key: "",
  created_by: "u-1",
  created_at: "",
  updated_at: "",
  can_run: true,
};

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-mine",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-mine",
  project_id: "p-1",
  conversation_id: "c-mine",
  starter_id: "u-1",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "done",
  started_at: "2026-09-17T10:00:00Z",
  ended_at: "2026-09-17T10:05:00Z",
  last_error: "",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

const conversation: Conversation = {
  id: "c-mine",
  workspace_id: "ws-1",
  kind: "ticket_thread",
  ticket_id: "t-mine",
  name: "",
  created_by: "u-1",
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
};

const stepFrame = (step: number, own: boolean): RunFrame => ({
  trail_id: own ? "tr-mine" : "tr-other",
  play_id: "play-1",
  target_type: "ticket",
  target_id: own ? "t-mine" : "t-other",
  state: "running",
  activity: { at: `2026-09-17T11:${String(step).padStart(2, "0")}:00Z`, kind: "tool_call", summary: `Step ${step}`, call_id: `call-${step}` } as RunFrame["activity"],
  ended_at: null,
  last_error: "",
});

const runSteps = (steps: number, own: boolean) => {
  for (let i = 0; i < steps; i += 1) act(() => usePlayRunStore.getState().applyFrame(stepFrame(i, own)));
};

const commitsDuring = async (ui: React.ReactElement, ready: () => Promise<unknown>, own = false) => {
  let commits = 0;
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <Profiler id="subject" onRender={() => (commits += 1)}>
        {ui}
      </Profiler>
    </QueryClientProvider>,
  );
  await ready();
  await act(async () => {});
  commits = 0;
  runSteps(60, own);
  return commits;
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions: ["plays:run"] } };
    if (url === "/api/auth/me") return { data: { user: { id: "u-me", login: "me" } } };
    if (url === "/api/plays/runs") return { data: [trail({})] };
    if (url === "/api/pairing/presence") return { data: { computers: {} } };
    return { data: [] };
  });
});

const button = <PlayButton play={play} projectId="p-1" targetType="ticket" targetId="t-mine" />;
const buttonReady = () => screen.findByRole("button", { name: "Fix with AI" });

const thread = (
  <MessageList conversation={conversation} messages={[]} currentUserId="u-1" resolveAuthor={unknownPerson} onEdit={async () => {}} onDelete={async () => {}} onInterruptAgent={() => {}} />
);
const threadReady = () => waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/runs", { params: { target_type: "ticket", target_id: "t-mine" } }));

describe("a 60-step run on another ticket", () => {
  it("commits no play button of this ticket", async () => {
    expect(await commitsDuring(button, buttonReady)).toBe(0);
  });

  it("commits no thread of this ticket", async () => {
    expect(await commitsDuring(thread, threadReady)).toBe(0);
  });
});

describe("a 60-step run on this ticket", () => {
  it("still commits its play button", async () => {
    expect(await commitsDuring(button, buttonReady, true)).toBeGreaterThan(0);
  });

  it("still commits its thread", async () => {
    expect(await commitsDuring(thread, threadReady, true)).toBeGreaterThan(0);
  });
});
