import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MessageReactions } from "@/components/chat/MessageReactions";
import { getMeKey } from "@/hooks/AuthHooks";
import { getChatMessagesKey } from "@/hooks/ChatHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { api } from "@/api/client";
import type { Message } from "@/models/Chat";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), put: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const message: Message = {
  id: "m1",
  conversation_id: "c1",
  author_id: "u2",
  author_kind: "user",
  body: "shipped",
  mentions: null,
  created_at: "2026-10-03T10:00:00Z",
  updated_at: "2026-10-03T10:00:00Z",
  reactions: [
    { emoji: "👍", user_ids: ["u2", "u1"] },
    { emoji: "🎉", user_ids: ["u2"] },
  ],
};

const renderReactions = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData([getMeKey], { user: { id: "u1" } });
  client.setQueryData([getChatMessagesKey, "c1", undefined], [message]);
  render(
    <QueryClientProvider client={client}>
      <MessageReactions message={message} />
    </QueryClientProvider>,
  );
  return client;
};

describe("MessageReactions", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockResolvedValue({ data: { people: [{ user_id: "u2", login: "sam", display_name: "Sam", avatar_url: "" }] } });
    vi.mocked(api.put).mockReturnValue(new Promise(() => {}));
    useWorkspaceStore.setState({ selectedWorkspaceId: "w1" });
  });

  it("names who reacted and marks the chips that are yours", async () => {
    renderReactions();
    expect(await screen.findByRole("button", { name: /👍 by Sam/ })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "🎉 by Sam" })).toHaveAttribute("aria-pressed", "false");
  });

  it("a chip takes back your reaction or adds it, patching the thread before the server answers", async () => {
    const client = renderReactions();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /👍/ }));
    await user.click(screen.getByRole("button", { name: /🎉/ }));
    expect(api.put).toHaveBeenCalledWith("/api/chat/messages/m1/reactions", { emoji: "👍", reacted: false });
    expect(api.put).toHaveBeenCalledWith("/api/chat/messages/m1/reactions", { emoji: "🎉", reacted: true });
    expect(client.getQueryData<Message[]>([getChatMessagesKey, "c1", undefined])?.[0]?.reactions).toEqual([
      { emoji: "👍", user_ids: ["u2"] },
      { emoji: "🎉", user_ids: ["u2", "u1"] },
    ]);
  });
});
