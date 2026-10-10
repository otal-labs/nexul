import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DecisionsCheckNotice } from "@/components/play/DecisionsCheckNotice";
import { DECISIONS_CHECK_KEY } from "@/models/Play";
import type { Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

const CHECK_ID = "play-check";

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: CHECK_ID,
  play_label: "Decisions check",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "",
  starter_id: "",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "failed",
  started_at: "2026-09-24T10:00:00Z",
  ended_at: "2026-09-24T10:00:00Z",
  last_error: "the provider is not confirmed on this computer",
  failure_reason: "",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

// The viewer's run buttons on the done ticket; without the seeded check among them there is nothing to retry.
let applicable = [{ id: CHECK_ID, label: "Decisions check", builtin_key: DECISIONS_CHECK_KEY }];

const serve = (trails: Trail[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/tickets/t-1") return { data: { id: "t-1", project_id: "p-1", status: "st-done" } };
    if (url === "/api/statuses") return { data: [{ id: "st-done", kind: "done" }] };
    if (url === "/api/workspaces/ws-1/plays/applicable") return { data: applicable };
    return { data: trails };
  });

const renderNotice = (trails: Trail[]) => {
  serve(trails);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DecisionsCheckNotice ticketId="t-1" />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  usePlayRunStore.setState({ frames: {}, steps: {}, activeByTarget: {} });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  applicable = [{ id: CHECK_ID, label: "Decisions check", builtin_key: DECISIONS_CHECK_KEY }];
});

describe("DecisionsCheckNotice", () => {
  it("says the check didn't run, why, and runs it again on the viewer's computer", async () => {
    const started = trail({ id: "tr-2", state: "starting", last_error: "" });
    vi.mocked(api.post).mockResolvedValue({ data: started });
    renderNotice([trail({}), trail({ id: "tr-fix", play_id: "play-fix", play_label: "Fix with AI", state: "done" })]);

    expect(await screen.findByText("Decisions check didn't run")).toBeInTheDocument();
    expect(screen.getByText("the provider is not confirmed on this computer")).toBeInTheDocument();
    serve([started, trail({})]);
    await userEvent.setup().click(screen.getByRole("button", { name: "Run check" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/decisions-check", { ticket_id: "t-1" });
    await waitFor(() => expect(screen.queryByText("Decisions check didn't run")).not.toBeInTheDocument());
  });

  it("stays hidden when the latest check ran, even after an earlier failure", async () => {
    const { container } = renderNotice([trail({ id: "tr-2", state: "done", last_error: "" }), trail({})]);
    await waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("stays hidden when the viewer may not run the check on this ticket", async () => {
    applicable = [];
    const { container } = renderNotice([trail({})]);
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/plays/applicable", expect.anything()));
    expect(container).toBeEmptyDOMElement();
  });

  it("stays hidden on a ticket that never had a check", async () => {
    const { container } = renderNotice([trail({ play_id: "play-fix", play_label: "Fix with AI" })]);
    await waitFor(() => expect(api.get).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it("hides as soon as a live frame says the check is running again", async () => {
    renderNotice([trail({})]);
    expect(await screen.findByText("Decisions check didn't run")).toBeInTheDocument();
    usePlayRunStore.getState().applyFrame({
      trail_id: "tr-1", play_id: CHECK_ID, target_type: "ticket", target_id: "t-1",
      state: "running", activity: null, ended_at: null, last_error: "",
    });
    await waitFor(() => expect(screen.queryByText("Decisions check didn't run")).not.toBeInTheDocument());
  });
});
