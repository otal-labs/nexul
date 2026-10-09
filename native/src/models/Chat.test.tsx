import type { Conversation } from "@nexul/client-core/chat";

import { groupConversations } from "@/models/Chat";

const conversation = (overrides: Partial<Conversation>): Conversation => ({
  id: "c1",
  workspace_id: "w1",
  kind: "channel",
  created_by: "u1",
  created_at: "2026-09-28T10:00:00Z",
  updated_at: "2026-09-28T10:00:00Z",
  ...overrides,
});

describe("groupConversations", () => {
  test("voice channels and interview threads are left out and empty groups dropped", () => {
    const groups = groupConversations([
      conversation({ id: "a", name: "general" }),
      conversation({ id: "b", kind: "voice_channel" }),
      conversation({ id: "c", kind: "interview_thread" }),
      conversation({ id: "d", kind: "doc_thread" }),
    ]);
    expect(groups.map((g) => [g.title, g.conversations.map((c) => c.id)])).toEqual([
      ["Channels", ["a"]],
      ["Threads", ["d"]],
    ]);
  });
});
