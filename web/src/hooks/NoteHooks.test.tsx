import { describe, expect, it } from "vitest";

import { noteFollower } from "@/hooks/NoteHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the note follower", () => {
  it("refetches a note's text and its thread's files when the note's message changes, and nothing for a plain message", async () => {
    const client = seeded([
      [["getNoteText", "f-1"], "plan"],
      [["getAttachments", { conversation_id: "c-1" }], []],
    ]);
    const message = { id: "m-1", conversation_id: "c-1", author_id: "u-1", author_kind: "agent", body: "plan", mentions: null, created_at: "", updated_at: "" };
    await followFrame(noteFollower, "chat.message.updated", { message }, client);
    expect(isStale(client, ["getNoteText", "f-1"])).toBe(false);

    await followFrame(noteFollower, "chat.message.updated", { message: { ...message, attachment_id: "f-1" } }, client);
    expect([isStale(client, ["getNoteText", "f-1"]), isStale(client, ["getAttachments", { conversation_id: "c-1" }])]).toEqual([true, true]);
  });
});
