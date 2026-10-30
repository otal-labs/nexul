import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketAttachmentsSection } from "@/components/ticket/TicketAttachmentsSection";
import { getMyRoleKey } from "@/hooks/WorkspaceHooks";
import type { Attachment } from "@/models/Attachment";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(() => "error"),
}));

const own: Attachment = {
  id: "a-1",
  ticket_id: "t-1",
  name: "shot.png",
  content_type: "image/png",
  size: 2048,
  uploaded_by: "user-1",
  created_at: "2026-09-01T12:00:00Z",
};

const post = (id: string, extra: Record<string, unknown> = {}) => ({
  id, conversation_id: "c-1", author_id: "u-1", author_kind: "agent", body: id, mentions: null, created_at: "2026-09-02T12:00:00Z", updated_at: "", ...extra,
});

const arrange = (opts: { attachments?: Attachment[]; messages?: unknown[]; thread?: boolean }) => {
  const { attachments = [], messages = [], thread = true } = opts;
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/attachments") return { data: attachments };
    if (url === "/api/chat/tickets/thread-status") return { data: { "t-1": thread } };
    if (url === "/api/chat/conversations/c-1/messages") return { data: messages };
    return { data: new Blob(["png"]) };
  });
  vi.mocked(api.post).mockResolvedValue({ data: { id: "c-1", workspace_id: "ws-1", kind: "thread", ticket_id: "t-1" } });
  globalThis.URL.createObjectURL = vi.fn(() => "blob:shot");
  globalThis.URL.revokeObjectURL = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMyRoleKey, "ws-1"], { role_name: "Member", permissions: ["tickets:write"] });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/tickets/t-1"]}>
        <TicketAttachmentsSection ticketId="t-1" projectId="p-1" workspaceId="ws-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("TicketAttachmentsSection", () => {
  it("points to the files the thread holds in place of No files yet, and says how many", async () => {
    arrange({ messages: [post("m1", { attachment_id: "a-9" }), post("m2", { attachment_id: "a-8" }), post("m3")] });

    const link = await screen.findByRole("link", { name: "2 files in the thread" });
    expect(link).toHaveAttribute("href", "/tickets/t-1/thread");
    expect(screen.queryByText("No files yet.")).not.toBeInTheDocument();
  });

  it("says one file in the singular and skips a file whose message was deleted", async () => {
    arrange({ messages: [post("m1", { attachment_id: "a-9" }), post("m2", { attachment_id: "a-8", deleted_at: "2026-09-03T12:00:00Z" })] });

    expect(await screen.findByRole("link", { name: "1 file in the thread" })).toBeInTheDocument();
  });

  it("lists the ticket's own files first and the line to the thread after them", async () => {
    arrange({ attachments: [own], messages: [post("m1", { attachment_id: "a-9" })] });

    const file = await screen.findByText("shot.png");
    const link = await screen.findByRole("link", { name: "1 file in the thread" });
    expect(file.compareDocumentPosition(link) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("keeps No files yet when the thread holds none, and never opens a thread that does not exist", async () => {
    arrange({ thread: false });

    expect(await screen.findByText("No files yet.")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });
});
