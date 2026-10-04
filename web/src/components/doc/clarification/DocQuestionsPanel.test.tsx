import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocQuestionsPanel } from "@/components/doc/clarification/DocQuestionsPanel";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Clarification, ClarificationQuestion, ClarificationRound } from "@/models/DocClarification";
import type { Doc } from "@/models/Doc";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "failed"),
}));

const doc: Doc = {
  id: "doc-1", project_id: "p-1", folder_id: "f-1", title: "Booking app", body: "", version: 1,
  archived: false, locked: false, created_by: "u-1", created_at: "2026-10-01T00:00:00Z", updated_at: "2026-10-01T00:00:00Z",
};

const question = (id: string, round: number, answer: Partial<ClarificationQuestion> = {}): ClarificationQuestion => ({
  id, doc_id: "doc-1", round, position: 1, question: `Question ${id}?`, why: "The doc asks for a reminder.",
  options: [{ label: "Text message (Suggested)" }, { label: "Email" }], multi_select: false,
  selected: [], text: "", skipped: false, ...answer,
});

const round = (n: number, questions: ClarificationQuestion[], extra: Partial<ClarificationRound> = {}): ClarificationRound => ({
  doc_id: "doc-1", round: n, started_by: "u-dev", trail_id: `tr-${n}`, started_at: "2026-10-01T00:00:00Z", running: false,
  anything_else: "", anything_else_reply: "", questions, ...extra,
});

const waiting: Clarification = {
  rounds: [round(1, [question("q1", 1, { selected: ["Email"], answered_at: "2026-10-01T01:00:00Z" }), question("q2", 1)])],
  running: false, closed: false, can_close: false,
};

const play = (id: string, key: string, label: string) => ({
  id, workspace_id: "ws-1", label, type: "doc", description: "", instructions: "", enabled: true, show_when_stage: null,
  excluded_project_ids: [], builtin_key: key, created_by: "", created_at: "", updated_at: "",
});

const client_ = ["docs:read", "docs:write"];
const developer = ["docs:read", "docs:write", "docs:thread", "plays:run"];

let permissions: string[] = [];
let shownDoc: Doc = doc;

const serve = (clarification: Clarification) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/docs/doc-1/clarification") return { data: clarification };
    if (url === "/api/docs/doc-1") return { data: shownDoc };
    if (url === "/api/workspaces/ws-1/plays/applicable") {
      return { data: [play("pl-c", "clarify", "Clarify via AI"), play("pl-t", "to-tickets-via-ai", "To tickets via AI")] };
    }
    if (url === "/api/pairing/presence") return { data: { computers: {} } };
    if (url === "/api/pairing/resolve") return { data: { ok: false, reason: "unpaired" } };
    return { data: [] };
  });

const renderPanel = (held: string[], shown: Doc = doc) => {
  permissions = held;
  shownDoc = shown;
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { id: "u-1", name: "Alice" } });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: held });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DocQuestionsPanel docId={shown.id} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.put).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("DocQuestionsPanel", () => {
  it("shows a client the waiting questions with no agent wording and nothing picked for them", async () => {
    serve(waiting);
    const { container } = renderPanel(client_);

    expect(await screen.findByText("1 question waiting")).toBeInTheDocument();
    expect(screen.getByText("Question q2?")).toBeInTheDocument();
    expect(screen.getByText("Question 2 of 2")).toBeInTheDocument();
    expect(screen.getByText("Why we're asking:")).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Text message (Suggested)" })).not.toBeChecked();
    expect(screen.getByLabelText("Anything else?")).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/\bAI\b|agent/i);
    expect(screen.queryByRole("button", { name: /Clarify via AI/ })).not.toBeInTheDocument();
  });

  it("still takes answers on a locked doc, saving the pick and opening the next question", async () => {
    const user = userEvent.setup();
    serve(waiting);
    vi.mocked(api.put).mockResolvedValue({ data: question("q2", 1, { selected: ["Email"] }) });
    renderPanel(client_, { ...doc, locked: true });

    await user.click(await screen.findByRole("radio", { name: "Email" }));
    await user.click(screen.getByRole("button", { name: "Next" }));

    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/questions/q2", { selected: ["Email"], text: "", skipped: false }),
    );
  });

  it("skips a question, and saves the round's Anything else? once it loses focus", async () => {
    const user = userEvent.setup();
    serve(waiting);
    vi.mocked(api.put).mockImplementation(async (url: string) =>
      url.endsWith("anything-else") ? { data: round(1, [], { anything_else: "Can we pick a groomer?" }) } : { data: question("q2", 1, { skipped: true }) },
    );
    renderPanel(client_);

    await user.click(await screen.findByRole("button", { name: "Skip" }));
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/questions/q2", { selected: [], text: "", skipped: true }),
    );
    await user.type(screen.getByLabelText("Anything else?"), "Can we pick a groomer?");
    await user.tab();
    await waitFor(() =>
      expect(api.put).toHaveBeenCalledWith("/api/docs/doc-1/clarification/rounds/1/anything-else", { text: "Can we pick a groomer?" }),
    );
  });

  it("clears a saved answer back to pending", async () => {
    const user = userEvent.setup();
    serve(waiting);
    vi.mocked(api.delete).mockResolvedValue({ data: question("q1", 1) });
    renderPanel(client_);

    await user.click(await screen.findByRole("button", { name: /Question q1\?/ }));
    await user.click(screen.getByRole("button", { name: "Clear answer" }));

    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/docs/doc-1/clarification/questions/q1"));
  });

  it("shows a reader without docs:write the rounds with nothing to open", async () => {
    serve(waiting);
    renderPanel(["docs:read"]);

    expect(await screen.findByText("Question q2?")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Question q2\?/ })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Anything else?")).not.toBeInTheDocument();
  });

  it("tells a client more questions are on the way while a round runs, and a developer which round runs", async () => {
    const running: Clarification = { ...waiting, running: true, rounds: [...waiting.rounds, round(2, [], { running: true })] };
    serve(running);
    const { unmount } = renderPanel(client_, { ...doc, locked: true });
    expect(await screen.findByText("More questions are on the way")).toBeInTheDocument();
    expect(screen.queryByText(/running/)).not.toBeInTheDocument();
    unmount();

    serve({ ...running, can_close: true });
    renderPanel(developer, { ...doc, locked: true });
    expect(await screen.findByText("Round 2 running")).toBeInTheDocument();
    expect(screen.getByText("· The doc is locked until it ends")).toBeInTheDocument();
  });

  it("offers a developer Clarify via AI as the primary action once a round is answered", async () => {
    serve({ ...waiting, can_close: true, rounds: [round(1, [question("q1", 1, { selected: ["Email"] })])] });
    renderPanel(developer);

    expect(await screen.findByText("Round 1 answered")).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: /Clarify via AI/ })).toBeInTheDocument();
  });

  it("lets a developer close after no gaps, and flags answers changed since the doc was written", async () => {
    const user = userEvent.setup();
    const noGaps: Clarification = {
      ...waiting,
      can_close: true,
      rounds: [waiting.rounds[0]!, round(2, [], { no_gaps_at: "2026-10-01T00:30:00Z" })],
    };
    serve(noGaps);
    vi.mocked(api.post).mockResolvedValue({ data: { ...noGaps, closed: true } });
    renderPanel(developer);

    expect(await screen.findByText("No gaps left")).toBeInTheDocument();
    expect(screen.getByText("1 answer changed since the doc was written")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/clarification/close"));
    expect(await screen.findByRole("button", { name: /To tickets via AI/ })).toBeInTheDocument();
  });

  it("shows a client a closed clarification as all answered, folded, with the earlier Anything else? and its reply", async () => {
    const closed: Clarification = {
      ...waiting,
      closed: true,
      rounds: [round(1, waiting.rounds[0]!.questions, { anything_else: "Can we pick a groomer?", anything_else_reply: "Asked in Round 2.", closed_at: "x" })],
    };
    serve(closed);
    renderPanel(client_);

    expect(await screen.findByText("All answered")).toBeInTheDocument();
    expect(screen.getByText("· 1 round")).toBeInTheDocument();
    const section = screen.getByRole("region", { name: "Round 1" });
    expect(within(section).getByRole("button", { name: /Round 1/ })).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Asked in Round 2.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /To tickets via AI/ })).not.toBeInTheDocument();
  });
});
