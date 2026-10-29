import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DocHistoryDialog } from "@/components/doc/DocHistoryDialog";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Doc } from "@/models/Doc";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

const doc: Doc = {
  id: "doc-1",
  project_id: "p-1",
  folder_id: "f-1",
  title: "Release checklist",
  body: "",
  version: 3,
  archived: false,
  locked: false,
  created_by: "u-1",
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-03T10:00:00Z",
};

const body = (...lines: string[]) =>
  JSON.stringify({ type: "doc", content: lines.map((t) => ({ type: "paragraph", content: [{ type: "text", text: t }] })) });

const versions: Record<string, { version: number; title: string; body: string }> = {
  "/api/docs/doc-1/versions/3": { version: 3, title: "Release checklist", body: body("Watch error rates for thirty minutes then promote.", "Close the ticket.") },
  "/api/docs/doc-1/versions/2": { version: 2, title: "Release checklist", body: body("Watch error rates for an hour then promote.", "Close the ticket.") },
  "/api/docs/doc-1/versions/1": { version: 1, title: "Release checklist", body: body("Close the ticket.") },
};

const history = {
  versions: [
    { version: 3, created_at: "2026-10-03T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: true },
    { version: 2, created_at: "2026-10-02T10:00:00Z", name: "Q3 freeze rules", editor_ids: ["u-1"], via: "mcp", current: false },
    { version: 1, created_at: "2026-10-01T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: false },
  ],
  changes: [],
};

const renderDialog = (onClose = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["docs:read", "docs:write"] });
  render(
    <QueryClientProvider client={client}>
      <DocHistoryDialog doc={doc} onClose={onClose} />
    </QueryClientProvider>,
  );
  return onClose;
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/docs/doc-1/history") return { data: history };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice Moreau", avatar_url: "" }] } };
    return { data: versions[url] };
  });
  vi.mocked(api.post).mockResolvedValue({ data: doc });
});

describe("DocHistoryDialog", () => {
  it("opens on the newest version with its changes marked and Restore disabled", async () => {
    renderDialog();

    expect((await screen.findByText("thirty minutes")).tagName).toBe("INS");
    expect(screen.getByText("an hour").tagName).toBe("DEL");
    expect(screen.getByRole("button", { name: "Restore this version" })).toBeDisabled();
    expect(screen.getAllByText("Alice Moreau")).toHaveLength(3);
  });

  it("compares a picked version with the one before it and restores it", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getByRole("button", { name: /Q3 freeze rules/ }));

    expect((await screen.findByText("Watch error rates for an hour then promote.")).tagName).toBe("INS");
    expect(screen.getByText(/via an agent/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Restore this version" }));

    expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/versions/2/restore");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("shows the version as it was, with no marks, once Show changes is off", async () => {
    const user = userEvent.setup();
    renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getByRole("switch", { name: "Show changes" }));

    expect(screen.getByText("Watch error rates for thirty minutes then promote.")).toBeInTheDocument();
    expect(screen.queryByText("an hour")).not.toBeInTheDocument();
  });

  it("shows the oldest kept version without marks, since nothing came before it", async () => {
    const user = userEvent.setup();
    renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getAllByRole("button", { name: /Alice Moreau/ }).at(-1)!);

    expect((await screen.findByText("Close the ticket.")).tagName).toBe("P");
    expect(screen.getByText("The oldest version kept")).toBeInTheDocument();
  });
});
