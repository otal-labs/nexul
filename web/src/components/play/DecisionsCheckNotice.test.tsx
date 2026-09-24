import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DecisionsCheckNotice } from "@/components/play/DecisionsCheckNotice";
import { DECISIONS_CHECK_PLAY_ID, type Trail } from "@/models/Trail";
import { usePlayRunStore } from "@/stores/playRunStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

const trail = (overrides: Partial<Trail>): Trail => ({
  id: "tr-1",
  workspace_id: "ws-1",
  play_id: DECISIONS_CHECK_PLAY_ID,
  play_label: "Decisions check",
  target_type: "ticket",
  target_id: "t-1",
  project_id: "p-1",
  conversation_id: "",
  starter_id: "",
  via: "web",
  selected_memory_ids: [],
  custom_instructions: "",
  move_to_status_id: "",
  computer_id: "",
  provider: "",
  model: "",
  harness_session_id: "",
  state: "failed",
  started_at: "2026-09-24T10:00:00Z",
  ended_at: "2026-09-24T10:00:00Z",
  last_error: "the provider is not confirmed on this computer",
  reply_message_id: "",
  activity: [],
  ...overrides,
});

const renderNotice = (trails: Trail[]) => {
  vi.mocked(api.get).mockResolvedValue({ data: trails });
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
});

describe("DecisionsCheckNotice", () => {
  it("says the check didn't run, why, and runs it again on the viewer's computer", async () => {
    const started = trail({ id: "tr-2", state: "starting", last_error: "" });
    vi.mocked(api.post).mockResolvedValue({ data: started });
    renderNotice([trail({}), trail({ id: "tr-fix", play_id: "play-fix", play_label: "Fix with AI", state: "done" })]);

    expect(await screen.findByText("Decisions check didn't run")).toBeInTheDocument();
    expect(screen.getByText("the provider is not confirmed on this computer")).toBeInTheDocument();
    vi.mocked(api.get).mockResolvedValue({ data: [started, trail({})] });
    await userEvent.setup().click(screen.getByRole("button", { name: "Run check" }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/decisions-check", { ticket_id: "t-1" });
    await waitFor(() => expect(screen.queryByText("Decisions check didn't run")).not.toBeInTheDocument());
  });

  it("stays hidden when the latest check ran, even after an earlier failure", async () => {
    const { container } = renderNotice([trail({ id: "tr-2", state: "done", last_error: "" }), trail({})]);
    await waitFor(() => expect(api.get).toHaveBeenCalled());
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
      trail_id: "tr-1", play_id: DECISIONS_CHECK_PLAY_ID, target_type: "ticket", target_id: "t-1",
      state: "running", activity: null, ended_at: null, last_error: "",
    });
    await waitFor(() => expect(screen.queryByText("Decisions check didn't run")).not.toBeInTheDocument());
  });
});
