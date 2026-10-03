import { describe, expect, it } from "vitest";

import {
  attachmentMarkdown,
  buildMentionCandidates,
  composeMessageBody,
  conversationLabel,
  conversationPlayTarget,
  findMentionTrigger,
  groupConversations,
  isContinuation,
  matchesMentionPrefix,
  splitMessageBody,
  type Conversation,
  type Message,
} from "@/models/Chat";
import type { Attachment } from "@/models/Attachment";
import { unknownPerson, type Person } from "@/models/Person";

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

describe("groupConversations", () => {
  it("splits channels, voice channels, DMs, and doc threads, dropping other thread kinds from the browsable list", () => {
    const list = [
      conversation({ id: "c1", kind: "channel" }),
      conversation({ id: "c2", kind: "dm" }),
      conversation({ id: "c3", kind: "ticket_thread" }),
      conversation({ id: "c4", kind: "channel_thread" }),
      conversation({ id: "c5", kind: "voice_channel" }),
      conversation({ id: "c6", kind: "doc_thread", doc_id: "doc-1" }),
    ];
    const { channels, voiceChannels, dms, docThreads } = groupConversations(list);
    expect(channels.map((c) => c.id)).toEqual(["c1"]);
    expect(voiceChannels.map((c) => c.id)).toEqual(["c5"]);
    expect(dms.map((c) => c.id)).toEqual(["c2"]);
    expect(docThreads.map((c) => c.id)).toEqual(["c6"]);
  });
});

const members: Person[] = [
  { user_id: "u1", login: "onik97", display_name: "Onik", avatar_url: "" },
  { user_id: "u2", login: "olive", display_name: "", avatar_url: "" },
];

describe("buildMentionCandidates", () => {
  it("always offers @Agent first, then every workspace member", () => {
    const candidates = buildMentionCandidates(members);
    expect(candidates).toEqual([
      { kind: "agent", handle: "Agent" },
      { kind: "user", handle: "onik97" },
      { kind: "user", handle: "olive" },
    ]);
  });
});

describe("matchesMentionPrefix", () => {
  it("matches case-insensitive prefixes, and everything on an empty query", () => {
    const candidate = { kind: "user" as const, handle: "onik97" };
    expect(matchesMentionPrefix(candidate, "")).toBe(true);
    expect(matchesMentionPrefix(candidate, "on")).toBe(true);
    expect(matchesMentionPrefix(candidate, "ON")).toBe(true);
    expect(matchesMentionPrefix(candidate, "nik")).toBe(false);
  });
});

describe("findMentionTrigger", () => {
  it("finds an in-progress mention at the start of the text", () => {
    expect(findMentionTrigger("@oni")).toEqual({ start: 0, query: "oni" });
  });

  it("finds an in-progress mention after whitespace", () => {
    expect(findMentionTrigger("hey @Age")).toEqual({ start: 4, query: "Age" });
  });

  it("does not trigger mid-word (no preceding whitespace)", () => {
    expect(findMentionTrigger("foo@bar")).toBeUndefined();
  });

  it("does not trigger once the caret has moved past the token", () => {
    expect(findMentionTrigger("@onik97 hello")).toBeUndefined();
  });

  it("returns no query text right after typing '@'", () => {
    expect(findMentionTrigger("@")).toEqual({ start: 0, query: "" });
  });

  it("returns undefined with no '@' at all", () => {
    expect(findMentionTrigger("hello there")).toBeUndefined();
  });
});

const attachment = (overrides: Partial<Attachment>): Attachment => ({
  id: "a1",
  name: "shot.png",
  content_type: "image/png",
  size: 100,
  uploaded_by: "u1",
  created_at: "2026-08-26T00:00:00Z",
  ...overrides,
});

describe("attachmentMarkdown", () => {
  it("renders the attachment as an image-markdown line pointing at the attachment path", () => {
    expect(attachmentMarkdown(attachment({ id: "a-9", name: "shot.png" }))).toBe("![shot.png](/api/attachments/a-9)");
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

describe("composeMessageBody", () => {
  it("trims the text and appends one image-markdown line per attachment", () => {
    const body = composeMessageBody("  hello  ", [attachment({ id: "a-1", name: "one.png" }), attachment({ id: "a-2", name: "two.png" })]);
    expect(body).toBe("hello\n![one.png](/api/attachments/a-1)\n![two.png](/api/attachments/a-2)");
  });

  it("produces images-only output when the text is empty", () => {
    expect(composeMessageBody("", [attachment({ id: "a-1", name: "one.png" })])).toBe("![one.png](/api/attachments/a-1)");
  });

  it("produces text-only output when there are no attachments", () => {
    expect(composeMessageBody("hello", [])).toBe("hello");
  });
});

describe("conversationPlayTarget", () => {
  it("maps an interview thread to its project's interview", () => {
    expect(conversationPlayTarget(conversation({ kind: "interview_thread", project_id: "p-1" }))).toEqual({
      type: "interview",
      id: "p-1",
    });
    expect(conversationLabel(conversation({ kind: "interview_thread", project_id: "p-1" }))).toBe("Interview");
  });

  it("maps nothing for a channel", () => {
    expect(conversationPlayTarget(conversation({ kind: "channel", name: "general" }))).toBeNull();
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

  it.each<[string, Message | undefined, Message, boolean]>([
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
  ])("a message that %s", (_name, prev, curr, expected) => {
    expect(isContinuation(prev, curr)).toBe(expected);
  });
});
