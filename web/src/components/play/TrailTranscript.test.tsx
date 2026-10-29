import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TrailTranscript } from "@/components/play/TrailTranscript";
import { getTrailKey } from "@/hooks/TrailHooks";
import type { ActivityEntry, Trail, TrailQuestion } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const question: TrailQuestion = {
  request_id: "req-1",
  questions: [{ id: "q1", text: "Proceed?", options: [{ label: "Yes" }, { label: "No" }] }],
  asked_at: "2026-09-18T10:00:04Z",
};

vi.mock("@/hooks/useCanActOnTrail", () => ({ useCanActOnTrail: () => true }));

const at = (s: number) => `2026-09-18T10:00:${String(s).padStart(2, "0")}Z`;

const thinking: ActivityEntry = { kind: "text", summary: "Reading the handler first.", detail: "Reading the handler first.", at: at(1) };
const read: ActivityEntry = { kind: "tool_result", call_id: "c-1", tool: "Read", summary: '{"file_path":"handler.go"}', detail: '{"input":{"file_path":"handler.go"},"result":{"content":"package x"}}', at: at(2) };
const failedTest: ActivityEntry = { kind: "tool_result", call_id: "c-2", tool: "Bash", summary: "go test ./... · failed", detail: '{"input":{"command":"go test ./..."}}', at: at(3) };
const asked: ActivityEntry = { kind: "question", call_id: "c-3", tool: "AskUserQuestion", summary: "Proceed?", at: at(4) };
const fixing: ActivityEntry = { kind: "text", summary: "Fixing the nil check.", detail: "Fixing the nil check.", at: at(6) };
const edit: ActivityEntry = { kind: "tool_result", call_id: "c-4", tool: "Edit", summary: "/src/handler.go", detail: '{"input":{"file_path":"/src/handler.go"}}', at: at(7) };
const reply: ActivityEntry = { kind: "text", summary: "Opened the PR.", detail: "Opened the PR: https://github.com/otal-labs/nexul/pull/51\n\nAll `go test` green.", at: at(50) };

const trail = (overrides: Partial<Trail>): Trail => ({
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
  custom_instructions: "Touch only the docs.",
  harness_session_id: "sess-1",
  state: "done",
  started_at: "2026-09-18T10:00:00Z",
  ended_at: "2026-09-18T10:00:50Z",
  last_error: "",
  failure_reason: "",
  reply_message_id: "reply-1",
  computer_id: "",
  provider: "",
  model: "",
  activity: [thinking, read, failedTest, asked, fixing, edit, reply],
  question: { ...question, answer: { answers: { q1: { selected: ["Yes"] } } } },
  ...overrides,
});

// live mirrors what TrailDetailBody hands down once a frame overrides the fetched row.
const renderTranscript = (t: Trail, live: { state: Trail["state"]; question: TrailQuestion | null } = { state: t.state, question: t.question ?? null }) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getTrailKey, t.id], t);
  return render(
    <QueryClientProvider client={client}>
      <TrailTranscript trail={t} steps={t.activity} state={live.state} question={live.question} />
    </QueryClientProvider>,
  );
};

beforeEach(() => usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} }));

describe("TrailTranscript as a conversation", () => {
  it("reads bubble, turn group, question card, answer bubble, second group, then the reply as prose, in order", async () => {
    const user = userEvent.setup();
    const { container } = renderTranscript(trail({}));

    const blocks = Array.from(container.firstElementChild!.children);
    const texts = blocks.map((b) => b.textContent ?? "");
    expect(texts[0]).toContain("Started Fix with AI");
    expect(texts[0]).toContain("Touch only the docs.");
    expect(texts[1]).toMatch(/^Worked for 3s · explored 1 file, ran 1 command/);
    expect(within(blocks[2] as HTMLElement).getByRole("group", { name: "Question from the Agent" })).toBeInTheDocument();
    expect(texts[3]).toBe("Answered: Yes");
    expect(texts[4]).toMatch(/^Worked for 44s · edited 1 file/);
    expect(texts[5]).toContain("Opened the PR");

    expect(screen.getByRole("link", { name: /otal-labs\/nexul#51/ })).toHaveAttribute("href", "https://github.com/otal-labs/nexul/pull/51");
    expect(screen.getByText("go test", { selector: "code" })).toBeInTheDocument();

    const group = within(blocks[1] as HTMLElement);
    expect(group.queryByText("Reading the handler first.")).not.toBeInTheDocument();
    await user.click(group.getByRole("button", { name: /Worked for 3s/ }));
    expect(group.getByText("Reading the handler first.")).toBeInTheDocument();
    expect(group.getByText("Explored 1 file")).toBeInTheDocument();
    expect(group.getByRole("button", { name: "handler.go" })).toBeInTheDocument();
    expect(group.getByText("Ran")).toBeInTheDocument();
    expect(group.getByText("go test ./...")).toBeInTheDocument();
    expect(group.getByText("failed")).toBeInTheDocument();
    expect(group.queryByText(/· failed/)).not.toBeInTheDocument();
  });

  it("a running trail keeps its group open with the newest row spinning, and no reply is split off yet", () => {
    renderTranscript(trail({ state: "running", ended_at: null, question: null, activity: [thinking, { ...read, kind: "tool_call" }] }));
    expect(screen.getByRole("button", { name: /^Reading handler\.go/ })).toBeInTheDocument();
    expect(screen.getByText("Exploring 1 file")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "running" })).toBeInTheDocument();
  });

  it("lists neighbouring commands under one count, each without the leading cd into the run's directory", async () => {
    const user = userEvent.setup();
    const root = "/home/dev/.t3/worktrees/nexul/fix-the-board";
    const command = (id: string, summary: string): ActivityEntry => ({ kind: "tool_result", call_id: id, tool: "Bash", summary, detail: "{}", at: at(3) });
    renderTranscript(
      trail({
        question: null,
        activity: [command("k-1", `cd ${root} && git status`), command("k-2", `cd ${root}/web && bun run lint · failed`), read, reply],
      }),
    );
    await user.click(screen.getByRole("button", { name: /^Worked/ }));
    expect(screen.getByText("Ran 2 commands")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "git status" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "cd web && bun run lint" })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("failed")).toBeInTheDocument();
  });

  it("legacy string steps render as plain rows in the group", async () => {
    const user = userEvent.setup();
    renderTranscript(trail({ question: null, activity: [{ kind: "other", summary: "Read main.go", at: "" }, { kind: "other", summary: "Bash go test", at: "" }] }));
    await user.click(screen.getByRole("button", { name: /^Worked/ }));
    expect(screen.getByText("Read main.go")).toBeInTheDocument();
    expect(screen.getByText("Bash go test")).toBeInTheDocument();
  });

  it("a runner's note is a muted line after the turn", () => {
    renderTranscript(trail({ question: null, activity: [read, reply, { kind: "note", summary: "Ticket is already in done; not moving it back to In review", at: at(51) }] }));
    expect(screen.getByText("Ticket is already in done; not moving it back to In review")).toHaveClass("italic");
    expect(screen.queryByRole("img", { name: "step" })).not.toBeInTheDocument();
  });
});

describe("TrailTranscript with a question", () => {
  const waiting = () => trail({ state: "waiting", ended_at: null, question, activity: [read, asked] });

  it("a waiting trail shows the card with the options and a Send answer button where the question was asked", () => {
    renderTranscript(waiting());
    expect(screen.getByRole("heading", { name: "Proceed?" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Yes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send answer" })).toBeInTheDocument();
    expect(screen.queryByRole("img", { name: "question" })).not.toBeInTheDocument();
  });

  it("once answered the card is read-only and the answer follows it as the starter's bubble", () => {
    renderTranscript(trail({ activity: [read, asked] }));
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
    expect(screen.getByText("→ Yes")).toBeInTheDocument();
    expect(screen.getByText("Answered: Yes")).toBeInTheDocument();
  });

  it("the live frame's waiting state and question win over the fetched row", () => {
    const t = trail({ state: "running", ended_at: null, question: null, activity: [] });
    usePlayRunStore.getState().applyFrame({
      trail_id: t.id, play_id: t.play_id, target_type: "ticket", target_id: t.target_id, state: "waiting",
      activity: null, question, ended_at: null, last_error: "",
    });
    renderTranscript(t, { state: "waiting", question });
    expect(screen.getByRole("button", { name: "Send answer" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Working/ })).not.toBeInTheDocument();
  });
});
