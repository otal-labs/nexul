import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react-native";

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

test("the hand-off shows the helper's prompt, each step and its reply", async () => {
  get.mockResolvedValue([reply([{ ...handoff, id: "h0", prompt: "Another helper's prompt" }, handoff])]);
  await renderHandoff();

  expect(await screen.findByText("Run the unit tests and report failures.")).toBeTruthy();
  expect(screen.queryByText("Another helper's prompt")).toBeNull();
  expect(screen.getByText("Shell")).toBeTruthy();
  expect(screen.getByText("bun run test")).toBeTruthy();
  expect(screen.getByText("Retried after a timeout")).toBeTruthy();
  expect(screen.getByText("All 212 tests pass.")).toBeTruthy();
  expect(get).toHaveBeenCalledWith(expect.stringContaining("/conversations/c1/messages"));
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
