import { describe, expect, it } from "vitest";

import { applyReaction, conversationLabel, isContinuation, splitMessageBody, upsertMessage, type ChatMessage, type Conversation } from "@nexul/client-core/chat";
import { unknownPerson, type Person } from "@nexul/client-core/person";

const conversation = (overrides: Partial<Conversation>): Conversation => ({
  id: "c1",
  workspace_id: "ws-1",
  kind: "channel",
  created_by: "u1",
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
  ...overrides,
});

describe("conversationLabel", () => {
  it("uses the channel name when present", () => {
    expect(conversationLabel(conversation({ kind: "channel", name: "general" }))).toBe("general");
  });

  it("falls back to a kind-based label for DMs, ticket threads, and doc threads", () => {
    expect(conversationLabel(conversation({ kind: "dm" }))).toBe("Direct message");
    expect(conversationLabel(conversation({ kind: "ticket_thread" }))).toBe("Ticket thread");
    expect(conversationLabel(conversation({ kind: "doc_thread" }))).toBe("Doc thread");
    expect(conversationLabel(conversation({ kind: "channel_thread" }))).toBe("Thread");
  });

  const people: Record<string, Person> = {
    "u-2": { user_id: "u-2", login: "olive", display_name: "", avatar_url: "" },
    "u-3": { user_id: "u-3", login: "sam", display_name: "Sam Hill", avatar_url: "" },
  };
  const resolvePerson = (id: string): Person => people[id] ?? unknownPerson(id);

  it("labels a DM with the other participants' names when given DM context", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-1", "u-2"] });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolvePerson })).toBe("olive");
  });

  it("joins multiple other participants for a group DM", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-1", "u-2", "u-3"] });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolvePerson })).toBe("olive, Sam Hill");
  });

  it("labels a self-DM with your own login and a (you) marker", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-2"] });
    expect(conversationLabel(dm, { currentUserId: "u-2", resolvePerson })).toBe("olive (you)");
  });

  it("falls back to 'Direct message' when no participant ids are present yet", () => {
    const dm = conversation({ kind: "dm" });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolvePerson })).toBe("Direct message");
  });
});

describe("splitMessageBody", () => {
  it("returns a single text segment for a body with no attachment images", () => {
    expect(splitMessageBody("hello team")).toEqual([{ kind: "text", text: "hello team" }]);
  });

  it("splits text around an attachment image line", () => {
    const body = "check this out\n![shot.png](/api/attachments/a-9)\nwhat do you think?";
    expect(splitMessageBody(body)).toEqual([
      { kind: "text", text: "check this out" },
      { kind: "image", src: "/api/attachments/a-9", alt: "shot.png" },
      { kind: "text", text: "what do you think?" },
    ]);
  });

  it("returns only image segments for an images-only body", () => {
    const body = "![one.png](/api/attachments/a-1)\n![two.png](/api/attachments/a-2)";
    expect(splitMessageBody(body)).toEqual([
      { kind: "image", src: "/api/attachments/a-1", alt: "one.png" },
      { kind: "image", src: "/api/attachments/a-2", alt: "two.png" },
    ]);
  });

  it("trims blank lines left over from a removed image line but keeps interior line breaks", () => {
    const body = "para one\nstill para one\n\n![shot.png](/api/attachments/a-9)\n\npara two";
    expect(splitMessageBody(body)).toEqual([
      { kind: "text", text: "para one\nstill para one" },
      { kind: "image", src: "/api/attachments/a-9", alt: "shot.png" },
      { kind: "text", text: "para two" },
    ]);
  });

  it("keeps markdown image syntax that isn't an attachment path as plain text", () => {
    const body = "![a diagram](https://example.com/diagram.png)";
    expect(splitMessageBody(body)).toEqual([{ kind: "text", text: body }]);
  });

  it("turns a one-line fence into a code block", () => {
    expect(splitMessageBody("```Hello World```")).toEqual([{ kind: "code", code: "Hello World" }]);
  });

  it("drops the language hint and keeps indentation and attachment lines inside a multi-line fence", () => {
    const body = "try this\n```go\nif err != nil {\n  return err\n}\n![shot.png](/api/attachments/a-9)\n```\nthen run it";
    expect(splitMessageBody(body)).toEqual([
      { kind: "text", text: "try this" },
      { kind: "code", code: "if err != nil {\n  return err\n}\n![shot.png](/api/attachments/a-9)" },
      { kind: "text", text: "then run it" },
    ]);
  });

  it("keeps code on the fence lines when they carry more than a language hint", () => {
    expect(splitMessageBody("```const a = 1;\nconst b = 2;```")).toEqual([{ kind: "code", code: "const a = 1;\nconst b = 2;" }]);
  });

  it("leaves an unclosed fence as plain text", () => {
    const body = "```go\nfmt.Println()";
    expect(splitMessageBody(body)).toEqual([{ kind: "text", text: body }]);
  });
});

describe("isContinuation", () => {
  const at = (hour: number, minute: number, second = 0, day = 28) => new Date(2026, 8, day, hour, minute, second).toISOString();
  const message = (overrides: Partial<ChatMessage>): ChatMessage => ({
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

  it.each<[string, ChatMessage | undefined, ChatMessage, boolean]>([
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
    ["is the same bot under the same name", message({ author_kind: "bot", author_id: "b1", author_name: "CI" }), message({ author_kind: "bot", author_id: "b1", author_name: "CI", created_at: at(10, 1) }), true],
    ["is the same bot posting as another name", message({ author_kind: "bot", author_id: "b1", author_name: "CI" }), message({ author_kind: "bot", author_id: "b1", author_name: "GitHub", created_at: at(10, 1) }), false],
    ["is the same bot with another avatar", message({ author_kind: "bot", author_id: "b1", author_name: "CI" }), message({ author_kind: "bot", author_id: "b1", author_name: "CI", author_avatar_url: "https://example.com/a.png", created_at: at(10, 1) }), false],
    ["is a bot after a person sharing its id", message({}), message({ author_kind: "bot", created_at: at(10, 1) }), false],
  ])("a message that %s", (_name, prev, curr, expected) => {
    expect(isContinuation(prev, curr)).toBe(expected);
  });
});

describe("applyReaction", () => {
  const thumbs = { emoji: "👍", user_ids: ["u1"] };
  it.each([
    ["adds a new emoji last", [thumbs], "🎉", "u2", true, [thumbs, { emoji: "🎉", user_ids: ["u2"] }]],
    ["joins an emoji already there", [thumbs], "👍", "u2", true, [{ emoji: "👍", user_ids: ["u1", "u2"] }]],
    ["an echoed add changes nothing", [thumbs], "👍", "u1", true, [thumbs]],
    ["removing the last person drops the emoji", [thumbs], "👍", "u1", false, []],
    ["removing a reaction nobody made changes nothing", [thumbs], "🎉", "u1", false, [thumbs]],
    ["a message with no reactions yet", undefined, "👍", "u1", true, [thumbs]],
  ])("%s", (_name, reactions, emoji, userId, reacted, want) => {
    expect(applyReaction(reactions, emoji, userId, reacted)).toEqual(want);
  });
});

describe("upsertMessage", () => {
  const sent = (overrides: Partial<ChatMessage>): ChatMessage => ({
    id: "m1",
    conversation_id: "c1",
    author_id: "u1",
    author_kind: "user",
    body: "hello",
    mentions: null,
    created_at: "2026-08-26T10:00:00Z",
    updated_at: "2026-08-26T10:00:00Z",
    ...overrides,
  });
  const pending = sent({ id: "pending-1", pending: true });

  it("retires the pending row its server copy confirms, and the copy keeps the row's key", () => {
    expect(upsertMessage([pending], sent({}))).toEqual([{ ...sent({}), client_key: "pending-1" }]);
  });

  it("keeps the key when the second copy (the POST reply after the push, or an edit) replaces the first", () => {
    const confirmed = upsertMessage([pending], sent({}));
    expect(upsertMessage(confirmed, sent({ body: "hello again", edited_at: "2026-08-26T10:01:00Z" }))).toEqual([
      { ...sent({ body: "hello again", edited_at: "2026-08-26T10:01:00Z" }), client_key: "pending-1" },
    ]);
  });

  it("leaves someone else's message with the same text and an unconfirmed row alone", () => {
    const other = sent({ id: "m2", author_id: "u2" });
    expect(upsertMessage([pending], other)).toEqual([pending, { ...other, client_key: undefined }]);
  });
});
