import { describe, expect, it } from "vitest";

import {
  attachmentMarkdown,
  buildMentionCandidates,
  composeMessageBody,
  conversationLabel,
  findMentionTrigger,
  groupConversations,
  matchesMentionPrefix,
  splitMessageBody,
  type Conversation,
} from "@/models/Chat";
import type { Attachment } from "@/models/Attachment";
import type { MemberView } from "@/models/Member";

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

  const resolveLogin = (id: string): string => ({ "u-2": "olive", "u-3": "sam" })[id] ?? id;

  it("labels a DM with the other participants' logins when given DM context", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-1", "u-2"] });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolveLogin })).toBe("olive");
  });

  it("joins multiple other participants for a group DM", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-1", "u-2", "u-3"] });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolveLogin })).toBe("olive, sam");
  });

  it("labels a self-DM with your own login and a (you) marker", () => {
    const dm = conversation({ kind: "dm", participant_ids: ["u-2"] });
    expect(conversationLabel(dm, { currentUserId: "u-2", resolveLogin })).toBe("olive (you)");
  });

  it("falls back to 'Direct message' when no participant ids are present yet", () => {
    const dm = conversation({ kind: "dm" });
    expect(conversationLabel(dm, { currentUserId: "u-1", resolveLogin })).toBe("Direct message");
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

const members: MemberView[] = [
  { user_id: "u1", login: "onik97", role_id: "r1" },
  { user_id: "u2", login: "olive", role_id: "r1" },
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
