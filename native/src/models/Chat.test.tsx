import { conversationLabel, groupConversations, isContinuation, splitMessageBody, type Conversation, type Message } from "@/models/Chat";

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

describe("isContinuation", () => {
  const at = (hour: number, minute: number, second = 0, day = 28) => new Date(2026, 8, day, hour, minute, second).toISOString();
  const message = (overrides: Partial<Message>): Message => ({
    id: "m",
    conversation_id: "c1",
    author_id: "u1",
    author_kind: "user",
    body: "hi",
    mentions: null,
    created_at: at(10, 0),
    updated_at: at(10, 0),
    ...overrides,
  });

  test.each<[string, Message | undefined, Message, boolean]>([
    ["is the first message", undefined, message({}), false],
    ["is the same person a minute later", message({}), message({ created_at: at(10, 1) }), true],
    ["is the same person exactly five minutes later", message({}), message({ created_at: at(10, 5) }), true],
    ["is the same person past five minutes", message({}), message({ created_at: at(10, 5, 1) }), false],
    ["chains: measured from the previous message, not the group's first", message({ created_at: at(10, 4) }), message({ created_at: at(10, 8) }), true],
    ["is a different person", message({}), message({ author_id: "u2", created_at: at(10, 1) }), false],
    ["is the Agent speaking as the same person", message({}), message({ author_kind: "agent", created_at: at(10, 1) }), false],
    ["follows an Agent message from the same person", message({ author_kind: "agent" }), message({ created_at: at(10, 1) }), false],
    ["follows a system line", message({ author_kind: "system" }), message({ created_at: at(10, 1) }), false],
    ["follows a deleted message", message({ deleted_at: at(10, 0, 30) }), message({ created_at: at(10, 1) }), false],
    ["crosses midnight within five minutes", message({ created_at: at(23, 58) }), message({ created_at: at(0, 1, 0, 29) }), false],
    ["arrives before the previous one", message({ created_at: at(10, 5) }), message({ created_at: at(10, 4) }), false],
    ["is the same bot under the same name", message({ author_kind: "bot", author_name: "GitHub" }), message({ author_kind: "bot", author_name: "GitHub", created_at: at(10, 1) }), true],
    ["is the same bot posting under another name", message({ author_kind: "bot", author_name: "GitHub" }), message({ author_kind: "bot", author_name: "Grafana", created_at: at(10, 1) }), false],
  ])("a message that %s", (_name, prev, curr, expected) => {
    expect(isContinuation(prev, curr)).toBe(expected);
  });
});
