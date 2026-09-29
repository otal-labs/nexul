import { conversationLabel, groupConversations, splitMessageBody, type Conversation } from "@/models/Chat";

const conversation = (overrides: Partial<Conversation>): Conversation => ({
  id: "c1",
  workspace_id: "w1",
  kind: "channel",
  created_by: "u1",
  created_at: "2026-09-28T10:00:00Z",
  updated_at: "2026-09-28T10:00:00Z",
  ...overrides,
});

describe("splitMessageBody", () => {
  test("a foreign image URL stays markdown text and never becomes an image", () => {
    expect(splitMessageBody("![x](https://evil.example/x.png)")).toEqual([
      { kind: "text", text: "![x](https://evil.example/x.png)" },
    ]);
  });

  test("an attachment line splits the text around it", () => {
    expect(splitMessageBody("look\n![shot.png](/api/attachments/a1)\n\nafter")).toEqual([
      { kind: "text", text: "look" },
      { kind: "image", src: "/api/attachments/a1", alt: "shot.png" },
      { kind: "text", text: "after" },
    ]);
  });
});

describe("conversationLabel", () => {
  const dmCtx = {
    currentUserId: "me",
    resolvePerson: (id: string) => ({ user_id: id, login: `@${id}`, display_name: id === "ana" ? "Ana Lima" : "", avatar_url: "" }),
  };

  test("a DM names the other people, not the caller", () => {
    expect(conversationLabel(conversation({ kind: "dm", participant_ids: ["me", "ana"] }), dmCtx)).toBe("Ana Lima");
  });

  test("a DM with only the caller says so", () => {
    expect(conversationLabel(conversation({ kind: "dm", participant_ids: ["me"] }), dmCtx)).toBe("@me (you)");
  });

  test("a named channel uses its name and an unnamed thread its kind", () => {
    expect(conversationLabel(conversation({ name: "general" }))).toBe("general");
    expect(conversationLabel(conversation({ kind: "ticket_thread" }))).toBe("Ticket thread");
  });
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
