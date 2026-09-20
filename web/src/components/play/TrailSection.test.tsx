import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TrailSection } from "@/components/play/TrailSection";
import type { ActivityEntry, Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

const readStep: ActivityEntry = {
  kind: "tool_result",
  call_id: "c-1",
  tool: "Read",
  summary: '{"file_path":"main.go"}',
  detail: '{"input":{"file_path":"main.go"},"result":{"type":"tool_result","content":"package main"}}',
  at: "2026-09-18T10:00:00Z",
};
const testStep: ActivityEntry = { kind: "tool_call", call_id: "c-2", tool: "Bash", summary: "go test", at: "2026-09-18T10:00:05Z" };
const replyStep: ActivityEntry = { kind: "text", summary: "Ran tests", detail: "Ran tests, all green.", at: "2026-09-18T10:00:09Z" };

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
  selected_memory_ids: ["m-always"],
  custom_instructions: "Keep the sensors as they are.",
  move_to_status_id: "st-review",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "s-1",
  state: "done",
  started_at: new Date(Date.now() - 3 * 60_000).toISOString(),
  ended_at: new Date().toISOString(),
  last_error: "",
  reply_message_id: "",
  activity: [readStep, replyStep],
  ...overrides,
});

const running = trail({ id: "tr-run", state: "running", ended_at: null, activity: [testStep], started_at: new Date().toISOString() });
const failed = trail({ id: "tr-fail", state: "failed", last_error: "Harness offline", starter_id: "u-2", activity: [] });

const mockApi = (trails: Trail[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/plays/runs") return { data: trails };
    if (url === "/api/plays/runs/tr-1") return { data: trails.find((t) => t.id === "tr-1") };
    if (url === "/api/plays/runs/tr-run") return { data: trails.find((t) => t.id === "tr-run") };
    if (url === "/api/workspaces/ws-1/members") return { data: { members: [{ user_id: "u-1", login: "onik97" }, { user_id: "u-2", login: "sara" }] } };
    if (url === "/api/memories") return { data: [{ id: "m-always", title: "Working in this project", always_included: true }] };
    if (url === "/api/statuses") return { data: [{ id: "st-review", name: "In review", kind: "review", position: 2 }] };
    return { data: [] };
  });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <TrailSection workspaceId="ws-1" targetType="ticket" targetId="t-1" />
    </QueryClientProvider>,
  );
};

const frameBase = { trail_id: "tr-run", play_id: "play-1", target_type: "ticket" as const, target_id: "t-1", ended_at: null, last_error: "" };

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
});

describe("TrailSection", () => {
  it("renders nothing when the ticket has no trails", async () => {
    mockApi([]);
    const { container } = renderSection();
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/plays/runs", expect.anything()));
    expect(container).toBeEmptyDOMElement();
  });

  it("lists one row per trail with its state, latest step, starter, and time", async () => {
    mockApi([running, trail({}), failed]);
    renderSection();

    expect(await screen.findByRole("heading", { name: "Trail" })).toBeInTheDocument();
    const rows = screen.getAllByRole("button");
    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent("Fix with AI · go test");
    await waitFor(() => expect(rows[0]).toHaveTextContent("onik97 · just now"));
    expect(rows[1]).toHaveTextContent("Fix with AI · Done");
    expect(rows[1]).toHaveTextContent("3m ago");
    expect(rows[2]).toHaveTextContent("Failed · Harness offline");
    expect(rows[2]).toHaveTextContent("sara");
    expect(screen.getByRole("img", { name: "running" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "done" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "failed" })).toBeInTheDocument();
  });

  it("opens the trail as a dialog: the run facts, the starter's message, the collapsed turn, and the reply as prose", async () => {
    const user = userEvent.setup();
    mockApi([trail({})]);
    renderSection();

    await user.click(await screen.findByRole("button", { name: /Fix with AI/ }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Working in this project");
    expect(dialog).toHaveTextContent("Keep the sensors as they are.");
    expect(dialog).toHaveTextContent("In review");
    await waitFor(() => expect(dialog).toHaveTextContent("Started byonik97"));

    expect(within(dialog).getByText("Started Fix with AI Keep the sensors as they are.", { normalizer: (s) => s.replace(/\s+/g, " ").trim() })).toBeInTheDocument();
    expect(within(dialog).getByText("Ran tests, all green.")).toBeInTheDocument();
    expect(within(dialog).queryByRole("listitem")).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: /Worked for 3m/ }));
    const steps = within(dialog).getAllByRole("listitem");
    expect(steps).toHaveLength(1);
    expect(steps[0]).toHaveTextContent('Read: {"file_path":"main.go"}');
    expect(steps[0]).toHaveTextContent("→package main");
    expect(within(steps[0]!).getByRole("img", { name: "tool result" })).toBeInTheDocument();

    expect(within(steps[0]!).getByText("Arguments")).not.toBeVisible();
    await user.click(within(steps[0]!).getByText('Read: {"file_path":"main.go"}'));
    expect(within(steps[0]!).getByText("Arguments")).toBeVisible();
    expect(steps[0]).toHaveTextContent('"file_path": "main.go"');
    expect(within(steps[0]!).getByText("Result")).toBeVisible();
  });

  it("follows live frames: the row's state and latest step change without a refetch", async () => {
    mockApi([running]);
    renderSection();
    expect(await screen.findByRole("img", { name: "running" })).toBeInTheDocument();

    act(() =>
      usePlayRunStore.getState().applyFrame({
        ...frameBase,
        state: "running",
        activity: { kind: "tool_call", call_id: "c-3", tool: "Bash", summary: "gh pr create", at: "2026-09-18T10:00:20Z" },
      }),
    );
    expect(screen.getByRole("button")).toHaveTextContent("Fix with AI · gh pr create");

    act(() => usePlayRunStore.getState().applyFrame({ ...frameBase, state: "interrupted", activity: null, last_error: "stopped by onik" }));
    expect(screen.getByRole("img", { name: "interrupted" })).toBeInTheDocument();
    expect(screen.getByRole("button")).toHaveTextContent("Interrupted");
  });

  it("an open dialog grows by the live step, replacing the call it completes", async () => {
    const user = userEvent.setup();
    mockApi([running]);
    renderSection();

    await user.click(await screen.findByRole("button", { name: /Fix with AI/ }));
    const dialog = await screen.findByRole("dialog");
    await waitFor(() => expect(within(dialog).getAllByRole("listitem")).toHaveLength(1));
    expect(within(within(dialog).getAllByRole("listitem")[0]!).getByRole("img", { name: "running" })).toBeInTheDocument();

    act(() =>
      usePlayRunStore.getState().applyFrame({
        ...frameBase,
        state: "running",
        activity: { ...testStep, kind: "tool_result", detail: '{"input":{"command":"go test"},"result":{"content":"ok"}}' },
      }),
    );
    const steps = within(dialog).getAllByRole("listitem");
    expect(steps).toHaveLength(1);
    expect(steps[0]).toHaveTextContent("→ok");
    expect(within(steps[0]!).getByRole("img", { name: "done" })).toBeInTheDocument();
  });
});
