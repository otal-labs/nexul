import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketHistoryDialog } from "@/components/ticket/TicketHistoryDialog";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

const ticket = { id: "t-1", project_id: "p-1", number: 7, title: "Write migrations" } as Ticket;

const body = (...lines: string[]) =>
  JSON.stringify({ type: "doc", content: lines.map((t) => ({ type: "paragraph", content: [{ type: "text", text: t }] })) });

// A ticket's version carries no title: the description is all it keeps.
const versions: Record<string, { version: number; body: string }> = {
  "/api/tickets/t-1/versions/2": { version: 2, body: body("Add the up and down scripts.") },
  "/api/tickets/t-1/versions/1": { version: 1, body: body("Add the scripts.") },
};

const actor = { kind: "user", user_id: "u-1" };
const history = {
  versions: [
    { version: 2, created_at: "2026-10-03T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: true },
    { version: 1, created_at: "2026-10-01T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: false },
  ],
  changes: [
    { id: "c-5", field: "labels", from: "", to: "security", actor: { kind: "play", user_id: "u-1", play_label: "Implement with AI" }, created_at: "2026-10-03T09:00:00Z" },
    { id: "c-4", field: "blocked_by", from: "", to: "t-2", actor, created_at: "2026-10-02T12:00:00Z" },
    { id: "c-3", field: "doc", from: "", to: "d-1", actor, created_at: "2026-10-02T11:00:00Z" },
    { id: "c-2", field: "developer", from: "", to: "sam", actor: { kind: "user:mcp", user_id: "u-1" }, created_at: "2026-10-02T10:00:00Z" },
    { id: "c-1", field: "created", from: "", to: "", actor, created_at: "2026-10-01T10:00:00Z" },
  ],
};

const renderDialog = (onClose = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["tickets:read", "tickets:write"] });
  render(
    <QueryClientProvider client={client}>
      <TicketHistoryDialog ticket={ticket} onClose={onClose} />
    </QueryClientProvider>,
  );
  return onClose;
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/tickets/t-1/history") return { data: history };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url === "/api/workspaces/ws-1/people") {
      return {
        data: {
          people: [
            { user_id: "u-1", login: "alice", display_name: "Alice Moreau", avatar_url: "" },
            { user_id: "u-2", login: "sam", display_name: "Sam Okafor", avatar_url: "" },
          ],
        },
      };
    }
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "BE" }] };
    if (url === "/api/tickets") return { data: [{ id: "t-2", project_id: "p-1", number: 3 }] };
    if (url === "/api/docs") return { data: [{ id: "d-1", title: "Migration plan" }] };
    return { data: versions[url] };
  });
  vi.mocked(api.post).mockResolvedValue({ data: ticket });
});

describe("TicketHistoryDialog", () => {
  it("compares a description that has no title and restores the picked version", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();

    expect((await screen.findByText("up and down")).tagName).toBe("INS");
    await user.click(screen.getAllByRole("button", { name: /Alice Moreau/ }).at(-1)!);
    await user.click(await screen.findByRole("button", { name: /^Restore / }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/versions/1/restore");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("lists field changes beside the versions, naming ids as the page does and who made each in its name", async () => {
    renderDialog();
    await screen.findByText("up and down");

    const rail = within(screen.getByRole("complementary", { name: "Versions" }));
    expect(await rail.findByText("security")).toBeInTheDocument();
    expect(rail.getByText("BE-3")).toBeInTheDocument();
    expect(rail.getByText("Migration plan")).toBeInTheDocument();
    expect(rail.getByText("Sam Okafor")).toBeInTheDocument();
    expect(rail.getByText(/Alice Moreau via Implement with AI/)).toBeInTheDocument();
    expect(rail.getByText(/Alice Moreau via an agent/)).toBeInTheDocument();
    expect(rail.getByText(/by Alice Moreau/)).toBeInTheDocument();
  });
});
