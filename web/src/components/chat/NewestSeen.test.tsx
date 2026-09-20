import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/components/ui/message-scroller", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, useMessageScrollerVisibility: () => ({ currentAnchorId: null, visibleMessageIds: ["m-2"] }) };
});

import { MessageList } from "@/components/chat/MessageList";
import type { Conversation, Message } from "@/models/Chat";

const conversation = { id: "c1", kind: "channel" } as Conversation;
const withClient = (ui: ReactNode) => render(<QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>);

const msg = (id: string): Message =>
  ({
    id,
    conversation_id: "c1",
    author_id: "u1",
    author_kind: "user",
    body: id,
    mentions: [],
    created_at: "2026-08-26T12:00:00Z",
    updated_at: "2026-08-26T12:00:00Z",
  }) as unknown as Message;

describe("MessageList onNewestSeen", () => {
  it("fires only when the newest message is actually visible", () => {
    const onSeen = vi.fn();
    withClient(
      <MessageList
        conversation={conversation}
        messages={[msg("m-1"), msg("m-2")]}
        currentUserId="u1"
        resolveAuthorLogin={() => "onik97"}
        onEdit={async () => {}}
        onDelete={async () => {}}
        onInterruptAgent={() => {}}
        onNewestSeen={onSeen}
      />,
    );
    expect(onSeen).toHaveBeenCalledWith("m-2");
  });

  it("does not fire when the newest message is offscreen", () => {
    const onSeen = vi.fn();
    withClient(
      <MessageList
        conversation={conversation}
        messages={[msg("m-2"), msg("m-3")]}
        currentUserId="u1"
        resolveAuthorLogin={() => "onik97"}
        onEdit={async () => {}}
        onDelete={async () => {}}
        onInterruptAgent={() => {}}
        onNewestSeen={onSeen}
      />,
    );
    expect(onSeen).not.toHaveBeenCalled();
  });
});
