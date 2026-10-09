import { describe, expect, it } from "vitest";

import { conversationLabel, type Conversation } from "@nexul/client-core/chat";
import type { Person } from "@nexul/client-core/person";

import { attachmentMarkdown, buildMentionCandidates, composeMessageBody, conversationPlayTarget, findMentionTrigger, groupConversations, matchesMentionPrefix } from "@/models/Chat";
import type { Attachment } from "@/models/Attachment";

const conversation = (overrides: Partial<Conversation>): Conversation => ({
  id: "c1",
  workspace_id: "ws-1",
  kind: "channel",
  created_by: "u1",
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
  ...overrides,
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

