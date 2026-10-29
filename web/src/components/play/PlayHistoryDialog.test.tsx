import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { PlayHistoryDialog } from "@/components/play/PlayHistoryDialog";
import { getMeKey } from "@/hooks/AuthHooks";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(),
}));

const bodies: Record<string, string> = {
  "/api/plays/play-1/versions/3": "Read the ticket and the files it names first.\n\nClose the ticket.\n\n- Post the notes in the channel",
  "/api/plays/play-1/versions/2": "Read the ticket and the files it names first.\n\nClose the ticket.",
  "/api/plays/play-1/versions/1": "Read the ticket first.\n\nClose the ticket.",
};

const history = () => ({
  versions: [
    { version: 3, created_at: "2026-10-29T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: true, builtin_default: false },
    { version: 2, created_at: "2026-10-28T10:00:00Z", name: "", editor_ids: ["u-1"], via: "", current: false, builtin_default: false },
    { version: 1, created_at: "2026-10-27T10:00:00Z", name: "", editor_ids: [], via: "", current: false, builtin_default: true },
  ],
  changes: [
    { id: "c-1", field: "enabled", from: "true", to: "false", actor: { kind: "user", user_id: "u-1" }, created_at: "2026-10-28T12:00:00Z" },
    { id: "c-2", field: "excluded_project_ids", from: "", to: "p-1", actor: { kind: "user", user_id: "u-1" }, created_at: "2026-10-28T13:00:00Z" },
  ],
});

const renderDialog = (version?: number, onClose = vi.fn()) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { name: "Alice" }, needs_owner_wizard: false, needs_first_login_wizard: false });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["plays:read", "plays:write"] });
  render(
    <QueryClientProvider client={client}>
      <PlayHistoryDialog playId="play-1" name="Fix with AI" {...(version !== undefined && { version })} onClose={onClose} />
    </QueryClientProvider>,
  );
  return onClose;
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/plays/play-1/history") return { data: history() };
    if (url === "/api/permissions/explain") return { data: { effective: [] } };
    if (url === "/api/projects") return { data: [{ id: "p-1", name: "Acme web" }] };
    if (url === "/api/workspaces/ws-1/people") return { data: { people: [{ user_id: "u-1", login: "alice", display_name: "Alice Moreau", avatar_url: "" }] } };
    return { data: url in bodies ? { version: Number(url.split("/").at(-1)), body: bodies[url] } : undefined };
  });
  vi.mocked(api.post).mockResolvedValue({ data: {} });
});

describe("PlayHistoryDialog", () => {
  it("opens on the newest instructions, marking the added list item, with field changes worded in the rail", async () => {
    renderDialog();

    expect(await screen.findByRole("group", { name: "Change 1 of 1, added" })).toHaveTextContent("Post the notes in the channel");
    expect(screen.getByText("Built-in default")).toBeInTheDocument();
    expect(screen.getByText("Enabled").parentElement).toHaveTextContent(/On\s*→\s*Off/);
    expect(await screen.findByText("Acme web")).toBeInTheDocument();
  });

  it("marks the words that changed in a picked version and restores it", async () => {
    const user = userEvent.setup();
    const onClose = renderDialog();
    await screen.findByRole("group", { name: "Change 1 of 1, added" });

    await user.click(screen.getAllByRole("button", { name: /Alice Moreau/ })[1]!);

    expect((await screen.findByText("and the files it names")).tagName).toBe("INS");
    await user.click(screen.getByRole("button", { name: /^Restore / }));
    expect(api.post).toHaveBeenCalledWith("/api/plays/play-1/versions/2/restore");
    await vi.waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it("opens on the version it is asked for, not the newest", async () => {
    renderDialog(2);

    expect((await screen.findByText("and the files it names")).tagName).toBe("INS");
    expect(screen.getByRole("button", { name: /Alice Moreau/, current: true })).toBeInTheDocument();
  });
});
