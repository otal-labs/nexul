import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { act } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import { MessageList } from "@/components/chat/MessageList";
import { getTrailsKey } from "@/hooks/TrailHooks";
import type { Conversation, Message } from "@/models/Chat";
import type { ActivityEntry, Trail } from "@/models/Trail";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { usePlayRunStore } from "@/stores/playRunStore";

const conversation: Conversation = {
  id: "c1",
  workspace_id: "ws-1",
  kind: "channel",
  name: "general",
  created_by: "u1",
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
};

const message = (overrides: Partial<Message>): Message => ({
  id: "m1",
  conversation_id: "c1",
  author_id: "u1",
  author_kind: "user",
  body: "hello",
  mentions: null,
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
  ...overrides,
});

const noopAsync = async () => {};

const renderList = (messages: Message[], onInterruptAgent = vi.fn()) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MessageList
        conversation={conversation}
        messages={messages}
        currentUserId="u1"
        resolveAuthorLogin={(id) => id}
        onEdit={noopAsync}
        onDelete={noopAsync}
        onInterruptAgent={onInterruptAgent}
      />
    </QueryClientProvider>,
  );

beforeEach(() => {
  useAgentStreamStore.setState({ streams: {} });
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
});

describe("MessageList agent stream bubble lifecycle", () => {
  it("renders no bubble when no stream frame exists for this conversation", () => {
    renderList([message({})]);
    expect(screen.queryByText("Agent")).not.toBeInTheDocument();
  });

  it("renders the in-progress bubble once a frame lands in the store, with a stop button while streaming", () => {
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "Looking into it", streaming: true }));
    renderList([message({})]);

    expect(screen.getByText("Agent")).toBeInTheDocument();
    expect(screen.getByText("Looking into it")).toBeInTheDocument();
    expect(screen.getByLabelText("Stop agent")).toBeInTheDocument();
  });

  it("full-text-replaces the bubble as later frames for the same conversation arrive", () => {
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "Looking", streaming: true }));
    renderList([message({})]);
    expect(screen.getByText("Looking")).toBeInTheDocument();

    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "Looking into it now", streaming: true }));
    expect(screen.queryByText("Looking")).not.toBeInTheDocument();
    expect(screen.getByText("Looking into it now")).toBeInTheDocument();
  });

  it("calls onInterruptAgent when the stop button is clicked", async () => {
    const onInterruptAgent = vi.fn();
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "hi", streaming: true }));
    renderList([message({})], onInterruptAgent);

    await userEvent.click(screen.getByLabelText("Stop agent"));
    expect(onInterruptAgent).toHaveBeenCalledOnce();
  });

  it("shows the latest tool step as a transcript row while no text has arrived", () => {
    act(() =>
      useAgentStreamStore.getState().setStream("c1", {
        messageId: "",
        text: "",
        streaming: true,
        activity: '{"file_path":"main.go"}',
        activityKind: "tool_call",
        activityTool: "Read",
      }),
    );
    renderList([message({})]);
    expect(screen.getByText(/Working for \d+s/)).toBeInTheDocument();
    expect(screen.getByText('Read: {"file_path":"main.go"}')).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "tool call" })).toBeInTheDocument();
  });

  it("hides the stop button once the frame stops streaming", () => {
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "done", streaming: false }));
    renderList([message({})]);
    expect(screen.getByText("done")).toBeInTheDocument();
    expect(screen.queryByLabelText("Stop agent")).not.toBeInTheDocument();
  });

  it("the ephemeral bubble yields to the persisted message once the store is cleared (final chat.message.created)", () => {
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "stream-1", text: "Working on it", streaming: true }));
    const { rerender } = renderList([message({})]);
    expect(screen.getByText("Working on it")).toBeInTheDocument();

    // The pipeline's persisted reply lands as a normal message, and useLiveEvents clears the store.
    act(() => useAgentStreamStore.getState().clearStream("c1"));
    const finalMessages = [message({}), message({ id: "m2", author_kind: "agent", body: "Here's the final answer." })];
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <MessageList
          conversation={conversation}
          messages={finalMessages}
          currentUserId="u1"
          resolveAuthorLogin={(id) => id}
          onEdit={noopAsync}
          onDelete={noopAsync}
          onInterruptAgent={vi.fn()}
        />
      </QueryClientProvider>,
    );

    expect(screen.queryByText("Working on it")).not.toBeInTheDocument();
    expect(screen.getByText("Here's the final answer.")).toBeInTheDocument();
  });

  it("ignores another conversation's stream frame", () => {
    act(() => useAgentStreamStore.getState().setStream("other-conversation", { messageId: "stream-1", text: "not mine", streaming: true }));
    renderList([message({})]);
    expect(screen.queryByText("not mine")).not.toBeInTheDocument();
  });
});

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const questionBody = "```nexul-question\n" + JSON.stringify({ request_id: "req-1", questions: [{ id: "q1", text: "Proceed?", options: [{ label: "Yes" }, { label: "No" }] }] }) + "\n```";

const renderWithClient = (messages: Message[]) =>
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MessageList
        conversation={conversation}
        messages={messages}
        currentUserId="u1"
        resolveAuthorLogin={(id) => id}
        onEdit={noopAsync}
        onDelete={noopAsync}
        onInterruptAgent={vi.fn()}
      />
    </QueryClientProvider>,
  );

describe("MessageList agent question", () => {
  it("renders the Agent's question message as the card", () => {
    renderWithClient([message({ id: "q", author_kind: "agent", body: questionBody })]);
    expect(screen.getByRole("heading", { name: "Proceed?" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Yes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send answer" })).toBeInTheDocument();
    expect(screen.queryByText(/nexul-question/)).not.toBeInTheDocument();
  });

  it("a question the thread already answered is read-only", () => {
    renderWithClient([
      message({ id: "q", author_kind: "agent", body: questionBody }),
      message({ id: "a", body: "Answered: Yes" }),
    ]);
    expect(screen.queryByRole("button", { name: "Send answer" })).not.toBeInTheDocument();
    expect(screen.getByText("Proceed?")).toBeInTheDocument();
  });
});

const ticketThread: Conversation = { ...conversation, kind: "ticket_thread", ticket_id: "t-1" };

const at = (sec: number) => `2026-09-18T10:00:${String(sec).padStart(2, "0")}Z`;
const read: ActivityEntry = { kind: "tool_result", call_id: "c-1", tool: "Read", summary: '{"file_path":"handler.go"}', at: at(2) };
const edit: ActivityEntry = { kind: "tool_result", call_id: "c-2", tool: "Edit", summary: "/src/handler.go", at: at(7) };
const reply: ActivityEntry = { kind: "text", summary: "Opened the PR.", detail: "Opened the PR: https://github.com/otal-labs/nexul/pull/51", at: at(30) };

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: "play-1",
  play_label: "Fix with AI",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "c1",
  starter_id: "u1",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  move_to_status_id: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "sess-1",
  state: "done",
  started_at: at(0),
  ended_at: at(30),
  last_error: "",
  reply_message_id: "reply-1",
  activity: [read, edit, reply],
  ...overrides,
});

const renderThread = (messages: Message[], trails: Trail[]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getTrailsKey, "ticket", "t-1"], trails);
  return render(
    <QueryClientProvider client={client}>
      <MessageList
        conversation={ticketThread}
        messages={messages}
        currentUserId="u1"
        resolveAuthorLogin={(id) => id}
        onEdit={noopAsync}
        onDelete={noopAsync}
        onInterruptAgent={vi.fn()}
      />
    </QueryClientProvider>,
  );
};

describe("MessageList play runs in a ticket thread", () => {
  it("the run's turn group sits above the reply it ended on, and the reply reads as prose with its PR chip", async () => {
    renderThread(
      [message({ body: "Started Fix with AI" }), message({ id: "reply-1", author_kind: "agent", body: "Opened the PR: https://github.com/otal-labs/nexul/pull/51" })],
      [trail({})],
    );
    expect(screen.getByText("Started Fix with AI")).toBeInTheDocument();
    const group = screen.getByRole("button", { name: /Worked for 30s/ });
    expect(screen.getByText("Used 2 tools")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /otal-labs\/nexul#51/ })).toHaveAttribute("href", "https://github.com/otal-labs/nexul/pull/51");
    expect(document.querySelector('[data-slot="bubble"]')).toHaveTextContent("Started Fix with AI");

    await userEvent.click(group);
    expect(screen.getByText('Read: {"file_path":"handler.go"}')).toBeInTheDocument();
    expect(screen.getByText("Edit: /src/handler.go")).toBeInTheDocument();
  });

  it("a running run's stream bubble shows the open turn group with every step the frames carried", () => {
    act(() => useAgentStreamStore.getState().setStream("c1", { messageId: "", text: "", streaming: true, activity: "/src/handler.go", activityKind: "tool_call", activityTool: "Edit" }));
    act(() =>
      usePlayRunStore.getState().applyFrame({
        trail_id: "tr-1", play_id: "play-1", target_type: "ticket", target_id: "t-1", state: "running",
        activity: { ...edit, kind: "tool_call" }, ended_at: null, last_error: "",
      }),
    );
    renderThread([message({ body: "Started Fix with AI" })], [trail({ state: "running", ended_at: null, reply_message_id: "", activity: [read] })]);
    expect(screen.getByRole("button", { name: /Working for/ })).toBeInTheDocument();
    const rows = screen.getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(rows[1]).toHaveTextContent("/src/handler.go");
    expect(screen.queryByText(/Working for \d+s/, { selector: "span" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Stop agent")).toBeInTheDocument();
  });

  it("a play's question is answered on its trail, so the card stays open until the trail says otherwise", () => {
    const asked = { request_id: "req-1", questions: [{ id: "q1", text: "Proceed?", options: [{ label: "Yes" }, { label: "No" }] }], asked_at: at(5) };
    renderThread(
      [message({ body: "Started Fix with AI" }), message({ id: "q", author_kind: "agent", body: questionBody })],
      [trail({ state: "waiting", ended_at: null, reply_message_id: "", activity: [read, { kind: "question", call_id: "c-3", tool: "AskUserQuestion", summary: "Proceed?", at: at(5) }], question: asked })],
    );
    expect(screen.getByRole("button", { name: /Worked for 2s/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send answer" })).toBeInTheDocument();
  });

  it("a run that failed before replying leaves its turns above the runner's note", () => {
    renderThread(
      [message({ body: "Started Fix with AI" }), message({ id: "n", author_kind: "system", body: "Run failed: no harness update for 15m" })],
      [trail({ state: "failed", reply_message_id: "", last_error: "no harness update for 15m", activity: [read, { kind: "note", summary: "Run failed: no harness update for 15m", at: at(30) }] })],
    );
    expect(screen.getByRole("button", { name: /Worked for/ })).toBeInTheDocument();
    expect(screen.getByText("Run failed: no harness update for 15m")).toHaveClass("italic");
  });
});
