import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { MessageRow } from "@/components/chat/MessageRow";
import { useSessionStore } from "@/stores/sessionStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Message } from "@/models/Chat";
import { unknownPerson } from "@/models/Person";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), delete: vi.fn() },
  resolveWSBase: vi.fn(() => "ws://test"),
  errorMessage: vi.fn(() => "error"),
}));

const confirmOpen = vi.fn();
vi.mock("@/hooks/useConfirmationDialog", () => ({
  useConfirmationDialog: () => ({ open: confirmOpen }),
}));

const note: Message = {
  id: "m1",
  conversation_id: "c1",
  author_id: "u1",
  author_kind: "agent",
  body: "Left the rollout plan in a note",
  mentions: null,
  attachment_id: "a1",
  created_at: "2026-10-02T10:00:00Z",
  updated_at: "2026-10-02T10:00:00Z",
};

const sockets: string[] = [];

class FakeWebSocket {
  constructor(url: string) {
    sockets.push(url);
  }
  send() {}
  close() {}
}

const routes = (permissions: string[]): Record<string, unknown> => ({
  "/api/attachments": [
    { id: "a1", conversation_id: "c1", name: "rollout.md", content_type: "text/markdown", size: 2048, uploaded_by: "u1", created_at: "" },
  ],
  "/api/attachments/a1": "# Rollout\n\nShip behind the flag first.",
  "/api/tickets/t1": { id: "t1", project_id: "p1" },
  "/api/workspaces/ws-1/me": { role_name: "Member", permissions },
  "/api/auth/me": { user: { id: "u2", name: "Lena", avatar_url: "" } },
});

const renderNote = (permissions: string[]) => {
  const table = routes(permissions);
  vi.mocked(api.get).mockImplementation(async (url: string) => ({ data: table[url] ?? [] }));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MessageRow message={note} author={unknownPerson("onik97")} isOwn={false} ticketId="t1" onEdit={vi.fn()} onDelete={vi.fn()} />
    </QueryClientProvider>,
  );
};

const openNote = async () => {
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: /rollout\.md/ }));
  return { user, dialog: await screen.findByRole("dialog", { name: note.body }) };
};

beforeEach(() => {
  sockets.length = 0;
  vi.mocked(api.get).mockReset();
  vi.mocked(api.delete).mockReset();
  confirmOpen.mockReset();
  vi.stubGlobal("WebSocket", FakeWebSocket);
  useSessionStore.setState({ token: "tok" });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "" });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("A note in the thread", () => {
  it("gives a reader the file's render with no editor, no delete, and no live room", async () => {
    renderNote(["tickets:read"]);
    const { dialog } = await openNote();

    expect(await screen.findByText("Ship behind the flag first.")).toBeInTheDocument();
    expect(dialog).toHaveTextContent("read only");
    expect(screen.queryByLabelText("Note")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /delete/i })).not.toBeInTheDocument();
    expect(sockets).toEqual([]);
  });

  it("gives a ticket writer the editor joined to the note's room, and a delete", async () => {
    renderNote(["tickets:read", "tickets:write"]);
    await openNote();

    expect(await screen.findByLabelText("Note")).toBeInTheDocument();
    await waitFor(() => expect(sockets).toHaveLength(1));
    expect(sockets[0]).toContain("/ws/collab/notes/m1?");
    expect(screen.getByRole("button", { name: /delete/i })).toBeInTheDocument();
  });

  it("deletes the note's message only once the writer confirms", async () => {
    renderNote(["tickets:read", "tickets:write"]);
    const { user } = await openNote();
    const remove = await screen.findByRole("button", { name: /delete/i });

    confirmOpen.mockResolvedValueOnce(false);
    await user.click(remove);
    expect(api.delete).not.toHaveBeenCalled();

    confirmOpen.mockResolvedValueOnce(true);
    await user.click(remove);
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/chat/messages/m1"));
  });
});
