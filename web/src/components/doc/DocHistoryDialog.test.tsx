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
  "/api/docs/doc-1/versions/3": {
    version: 3,
    title: "Release checklist",
    body: body("Watch error rates for thirty minutes then promote.", "Close the ticket.", "Post the notes in the channel."),
  },
  "/api/docs/doc-1/versions/2": { version: 2, title: "Release checklist", body: body("Watch error rates for an hour then promote.", "Close the ticket.") },
  "/api/docs/doc-1/versions/1": { version: 1, title: "Release checklist", body: body("Close the ticket.") },
};

const threeVersions = () => ({
  versions: [
    { version: 3, created_at: "2026-10-03T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: true },
    { version: 2, created_at: "2026-10-02T10:00:00Z", name: "Q3 freeze rules", editor_ids: ["u-1"], via: "mcp", current: false },
    { version: 1, created_at: "2026-10-01T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: false },
  ],
  changes: [],
});

let history = threeVersions();

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
  history = threeVersions();
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
    expect(screen.getByRole("button", { name: "Restore" })).toBeDisabled();
    expect(screen.getByText("This is the current version")).toBeInTheDocument();
    expect(screen.getAllByText("Alice Moreau")).toHaveLength(3);
  });

  it("compares a picked version with the one before it and restores it", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getByRole("button", { name: /Q3 freeze rules/ }));

    const added = await screen.findByRole("group", { name: "Change 1 of 1, added" });
    expect(added).toHaveTextContent("Watch error rates for an hour then promote.");
    expect(screen.getByText(/via an agent/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^Restore / }));

    expect(api.post).toHaveBeenCalledWith("/api/docs/doc-1/versions/2/restore");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("shows the version as it was, with no marks, once As it was is picked", async () => {
    const user = userEvent.setup();
    renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getByRole("radio", { name: "As it was" }));

    expect(screen.getByText("Watch error rates for thirty minutes then promote.")).toBeInTheDocument();
    expect(screen.queryByText("an hour")).not.toBeInTheDocument();
    expect(screen.getByText(/as it was/)).toBeInTheDocument();
  });

  it("shows the oldest kept version without marks or a stepper, since nothing came before it", async () => {
    const user = userEvent.setup();
    renderDialog();
    await screen.findByText("thirty minutes");

    await user.click(screen.getAllByRole("button", { name: /Alice Moreau/ }).at(-1)!);

    expect((await screen.findByText("Close the ticket.")).tagName).toBe("P");
    expect(screen.getByText("Oldest kept version")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next change" })).not.toBeInTheDocument();
  });

  it("steps through the changes with ] and [, moving focus to each one", async () => {
    const user = userEvent.setup();
    renderDialog();
    expect(await screen.findByText("1 of 2 changes")).toBeInTheDocument();

    await user.keyboard("]");

    expect(screen.getByText("2 of 2 changes")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Change 2 of 2, added" })).toHaveFocus();
    await user.keyboard("[[");
    expect(screen.getByText("1 of 2 changes")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Change 1 of 2, edited" })).toHaveFocus();
  });

  it("counts a changed title as change 1, ahead of the body's changes, and the keys reach it", async () => {
    const user = userEvent.setup();
    const base = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url: string) =>
      url === "/api/docs/doc-1/versions/3" ? { data: { ...versions[url], title: "Release checklist (final)" } } : base(url),
    );
    renderDialog();

    expect(await screen.findByText("1 of 3 changes")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Change 1 of 3, edited" })).toHaveTextContent("(final)");
    await user.keyboard("]");
    await user.keyboard("]");
    expect(screen.getByRole("group", { name: "Change 3 of 3, added" })).toHaveFocus();
    await user.keyboard("[[");
    await user.keyboard("[[");
    expect(screen.getByRole("group", { name: "Change 1 of 3, edited" })).toHaveFocus();
  });

  it("shows the size and the restore the server sends with a version, and nothing for a version it sends none for", async () => {
    Object.assign(history.versions[0]!, { words_added: 1234, words_removed: 21, restored_from: 1 });
    renderDialog();

    expect(await screen.findByText("+1.2k")).toBeInTheDocument();
    expect(screen.getByText("−21")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /1234 words added, 21 removed/ })).toBeInTheDocument();
    expect(screen.getAllByText(/^[+−]\d/)).toHaveLength(2);
    expect(screen.getByText(/Restored from/)).toBeInTheDocument();
    expect(screen.getByText("Restored")).toBeInTheDocument();
  });

  it("shrinks to a single panel with nothing to compare or restore when the item has one version", async () => {
    history = { versions: [{ ...history.versions[2]!, current: true }], changes: [] };
    renderDialog();

    expect(await screen.findByText(/One version, saved/)).toHaveTextContent("by Alice Moreau");
    expect(screen.queryByRole("button", { name: /^Restore / })).not.toBeInTheDocument();
    expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  });
});
