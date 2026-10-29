import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { HandoffScreen } from "@/components/chat/HandoffScreen";
import type { Handoff, Message } from "@/models/Chat";

jest.mock("expo-router", () => ({
  useLocalSearchParams: () => ({ id: "h1", conversationId: "c1", messageId: "r1" }),
  Stack: { Screen: () => null },
}));
jest.mock("@/stores/sessionStore", () => ({
  useSessionStore: Object.assign(() => "https://nexul.example.com", { getState: () => ({ host: "https://nexul.example.com" }) }),
  readSessionToken: () => "ses_abc",
}));
jest.mock("@/hooks/WorkspaceHooks", () => ({ useAreaAccess: () => () => true }));
jest.mock("react-native-enriched-markdown", () => jest.requireActual("react-native-enriched-markdown/jest"));
jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const get = jest.mocked(api.get);

const handoff: Handoff = {
  id: "h1",
  driver: "codex",
  model: "gpt-5.5",
  title: "Run the tests",
  prompt: "Run the unit tests and report failures.",
  state: "done",
  reply: "All 212 tests pass.",
  steps: [
    { kind: "tool_result", call_id: "t1", tool: "Shell", summary: "bun run test", at: "2026-10-04T10:00:00Z" },
    { kind: "note", summary: "Retried after a timeout", at: "2026-10-04T10:01:00Z" },
  ],
};

const reply = (handoffs?: Handoff[]): Message => ({
  id: "r1",
  conversation_id: "c1",
  author_id: "agent",
  author_kind: "agent",
  body: "Tests are green.",
  mentions: null,
  created_at: "2026-10-04T10:02:00Z",
  updated_at: "2026-10-04T10:02:00Z",
  ...(handoffs && { handoffs }),
});

const renderHandoff = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <HandoffScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => get.mockReset());

test("a reply without the hand-off says it is no longer there", async () => {
  get.mockResolvedValue([reply()]);
  await renderHandoff();

  expect(await screen.findByText("This hand-off is gone")).toBeTruthy();
});

const turn = (name: string) => screen.getByRole("button", { name: new RegExp(`^${name}`) });

test("a failed step marks its turn and reads as failed", async () => {
  const failed = { kind: "tool_result", call_id: "t2", tool: "Shell", summary: "/bin/bash -lc 'go test ./...' · failed", at: "2026-10-04T10:00:30Z" } as const;
  get.mockResolvedValue([reply([{ ...handoff, state: "failed", steps: [failed] }])]);
  await renderHandoff();

  expect(await screen.findByLabelText("A step failed")).toBeTruthy();
  await fireEvent.press(turn("Worked"));
  expect(screen.getByLabelText("Ran go test ./..., failed")).toBeTruthy();
});

test("a call that never got its result marks the ended turn unfinished", async () => {
  const call = { kind: "tool_call", call_id: "t3", tool: "Read", summary: '{"file_path":"main.go"}', at: "2026-10-04T10:00:00Z" } as const;
  get.mockResolvedValue([reply([{ ...handoff, state: "interrupted", steps: [call] }])]);
  await renderHandoff();

  expect(await screen.findByLabelText("Unfinished")).toBeTruthy();
});

test("the hand-off shows the helper's prompt, its steps folded under how long it worked, and its reply", async () => {
  get.mockResolvedValue([reply([{ ...handoff, id: "h0", prompt: "Another helper's prompt" }, handoff])]);
  await renderHandoff();

  expect(await screen.findByText("Run the unit tests and report failures.")).toBeTruthy();
  expect(screen.queryByText("Another helper's prompt")).toBeNull();
  expect(screen.getByText("Worked for 1m 0s")).toBeTruthy();
  expect(screen.getByText("Ran 1 command")).toBeTruthy();
  expect(screen.getByLabelText("Done")).toBeTruthy();
  expect(screen.queryByText("bun run test")).toBeNull();

  await fireEvent.press(turn("Worked for 1m 0s"));
  expect(screen.getByText("bun run test")).toBeTruthy();
  expect(screen.getByText("Retried after a timeout")).toBeTruthy();
  expect(screen.getByText("All 212 tests pass.")).toBeTruthy();
  expect(get).toHaveBeenCalledWith(expect.stringContaining("/conversations/c1/messages"));
});

test("a running turn is open on its steps, and a message between turns reads as prose", async () => {
  const message = { kind: "text", summary: "Tests compiled, running them now.", at: "2026-10-04T10:00:10Z" } as const;
  const running = { kind: "tool_call", call_id: "t4", tool: "Shell", summary: "bun run test", at: "2026-10-04T10:00:20Z" } as const;
  get.mockResolvedValue([reply([{ ...handoff, state: "running", reply: "", steps: [handoff.steps[0]!, message, running] }])]);
  await renderHandoff();

  expect(await screen.findByText("Tests compiled, running them now.")).toBeTruthy();
  expect(screen.getByText("Running bun run test")).toBeTruthy();
  expect(screen.getAllByText("bun run test")).toHaveLength(1);
});

test("neighbouring commands list under one count, without the leading cd into the run's directory", async () => {
  const root = "/home/dev/.t3/worktrees/nexul/run-the-tests";
  const command = (id: string, summary: string) => ({ kind: "tool_result", call_id: id, tool: "Shell", summary, at: "2026-10-04T10:00:00Z" }) as const;
  get.mockResolvedValue([reply([{ ...handoff, steps: [command("t5", `/bin/bash -lc "cd ${root} && git status"`), command("t6", `cd ${root}/web && bun run lint · failed`)] }])]);
  await renderHandoff();

  await fireEvent.press(await screen.findByRole("button", { name: /^Worked/ }));
  expect(screen.getAllByText("Ran 2 commands")).toHaveLength(2);
  expect(screen.getByLabelText("git status")).toBeTruthy();
  expect(screen.getByLabelText("cd web && bun run lint, failed")).toBeTruthy();
});

test.each([
  ["done", "No reply came back."],
  ["left_running", "Still running in T3 Code."],
] as const)("a %s hand-off with no reply says why", async (state, line) => {
  get.mockResolvedValue([reply([{ ...handoff, state, reply: "" }])]);
  await renderHandoff();

  expect(await screen.findByText(line)).toBeTruthy();
});

test("a running hand-off with no steps yet shows it is working", async () => {
  get.mockResolvedValue([reply([{ ...handoff, state: "running", reply: "", steps: [] }])]);
  await renderHandoff();

  expect(await screen.findByText("Working")).toBeTruthy();
  expect(screen.queryByText("No reply came back.")).toBeNull();
});
