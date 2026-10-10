import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Conversation } from "@nexul/client-core/chat";
import { unknownPerson } from "@nexul/client-core/person";

import { api } from "@/api/client";
import { MessageList } from "@/components/chat/MessageList";
import { getPlayQueueKey } from "@/hooks/PlayQueueHooks";
import type { Message } from "@/models/Chat";
import type { PlayQueueItem } from "@/models/PlayQueue";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { playQueue, queueItem } from "@/test/playQueueItem";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const thread: Conversation = {
  id: "c1",
  workspace_id: "ws-1",
  kind: "ticket_thread",
  ticket_id: "t-1",
  name: "",
  created_by: "u1",
  created_at: "2026-10-10T08:00:00Z",
  updated_at: "2026-10-10T08:00:00Z",
};

const message = (id: string, body: string, created_at: string): Message => ({
  id,
  conversation_id: "c1",
  author_id: "u1",
  author_kind: "user",
  body,
  mentions: null,
  created_at,
  updated_at: created_at,
});

const play = { id: "play-1", workspace_id: "ws-1", label: "Fix with AI", type: "ticket", description: "", show_when_stage: "progress" };

const mockApi = (permissions: string[]) =>
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/workspaces/ws-1/me") return { data: { role_name: "Member", permissions } };
    if (url === "/api/workspaces/ws-1/plays") return { data: [play] };
    // The run dialog's own choices stay loading: opening it is what this file checks.
    if (url === "/api/plays/latest-choices") return new Promise(() => {});
    return { data: [] };
  });

const renderThread = (items: PlayQueueItem[], messages: Message[]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  client.setQueryData([getPlayQueueKey, "ticket", "t-1"], playQueue({ items }));
  render(
    <QueryClientProvider client={client}>
      <MessageList
        conversation={thread}
        messages={messages}
        currentUserId="u1"
        resolveAuthor={unknownPerson}
        onEdit={async () => {}}
        onDelete={async () => {}}
        onInterruptAgent={vi.fn()}
      />
    </QueryClientProvider>,
  );
};

const skipped = queueItem({ id: "q-skip", status: "skipped", reason: "no longer unblocked", decided_at: "2026-10-10T09:05:00Z" });
const didnt = queueItem({
  id: "q-didnt",
  status: "didnt_run",
  reason: "nobody to run it on: the ticket has no developer",
  decided_at: "2026-10-10T09:15:00Z",
});

beforeEach(() => {
  vi.resetAllMocks();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
});

describe("MessageList auto play lines", () => {
  it("places skipped and didn't-run runs among the messages by time, in plain words", () => {
    mockApi([]);
    renderThread(
      [didnt, skipped, queueItem({ id: "q-wait", queued_at: "2026-10-10T09:20:00Z" })],
      [message("m1", "First", "2026-10-10T09:00:00Z"), message("m2", "Second", "2026-10-10T09:10:00Z")],
    );
    const order = [screen.getByText("First"), screen.getByText(/skipped: no longer unblocked/), screen.getByText("Second"), screen.getByText(/didn't run: nobody is the ticket's developer/)];
    for (let i = 1; i < order.length; i++) {
      expect(order[i - 1]!.compareDocumentPosition(order[i]!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    expect(screen.queryByText(/queued/)).not.toBeInTheDocument();
  });

  it("puts a line on the same day as the next message under that day's divider, and gives a later day its own", () => {
    mockApi([]);
    renderThread(
      [
        { ...skipped, decided_at: "2026-10-10T11:00:00Z" },
        { ...didnt, decided_at: "2026-10-11T12:00:00Z" },
      ],
      [message("m1", "First", "2026-10-09T12:00:00Z"), message("m2", "Second", "2026-10-10T12:00:00Z")],
    );
    const [day9, day10, day11] = screen.getAllByRole("separator");
    const order = [day9, screen.getByText("First"), day10, screen.getByText(/skipped:/), screen.getByText("Second"), day11, screen.getByText(/didn't run:/)];
    expect(order.every(Boolean)).toBe(true);
    for (let i = 1; i < order.length; i++) {
      expect(order[i - 1]!.compareDocumentPosition(order[i]!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    expect(screen.getAllByRole("separator")).toHaveLength(3);
  });

  it("shows the lines in a thread with no messages yet", () => {
    mockApi([]);
    renderThread([skipped], []);
    expect(screen.getByText(/skipped: no longer unblocked/)).toBeInTheDocument();
    expect(screen.queryByText("No messages yet. Say hello.")).not.toBeInTheDocument();
  });

  it("offers Run it on the play's latest didn't-run to a viewer who may run plays, through the run dialog", async () => {
    mockApi(["plays:run"]);
    const older = { ...didnt, id: "q-old", decided_at: "2026-10-10T09:01:00Z" };
    renderThread([didnt, older], [message("m1", "First", "2026-10-10T09:00:00Z")]);
    const run = await screen.findByRole("button", { name: "Run Fix with AI" });
    expect(screen.getAllByRole("button", { name: "Run Fix with AI" })).toHaveLength(1);
    await userEvent.click(run);
    expect(await screen.findByRole("dialog", { name: "Fix with AI" })).toBeInTheDocument();
  });

  it("hides Run it from a viewer who may not run plays", async () => {
    mockApi([]);
    renderThread([didnt], [message("m1", "First", "2026-10-10T09:00:00Z")]);
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.getByText(/didn't run/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Run Fix with AI" })).not.toBeInTheDocument();
  });
});
