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

  const note = (id: string, conversationId = "c-1") => ({
    id, conversation_id: conversationId, author_id: "u-1", author_kind: "agent", body: id, mentions: null, attachment_id: `f-${id}`, created_at: "", updated_at: "",
  });
  const notesOf = (client: ReturnType<typeof seeded>, ticketId: string) =>
    (client.getQueryData(["getTicketNotes", ticketId]) as { notes: { id: string; body: string }[] }).notes;

  it("patches a note into its ticket's list and out again, leaving other tickets' lists and plain messages alone", async () => {
    const client = seeded([
      [["getTicketNotes", "t-1"], { conversation_id: "c-1", notes: [note("m-1")] }],
      [["getTicketNotes", "t-2"], { conversation_id: "c-2", notes: [note("m-9", "c-2")] }],
    ]);
    await followFrame(noteFollower, "chat.message.created", { message: { ...note("m-plain"), attachment_id: undefined } }, client);
    await followFrame(noteFollower, "chat.message.created", { message: note("m-2") }, client);
    expect(notesOf(client, "t-1").map((n) => n.id)).toEqual(["m-2", "m-1"]);

    await followFrame(noteFollower, "chat.message.updated", { message: { ...note("m-1"), body: "edited" } }, client);
    expect(notesOf(client, "t-1").map((n) => n.body)).toEqual(["m-2", "edited"]);

    await followFrame(noteFollower, "chat.message.deleted", { conversation_id: "c-1", message_id: "m-2" }, client);
    expect(notesOf(client, "t-1").map((n) => n.id)).toEqual(["m-1"]);
    expect(notesOf(client, "t-2").map((n) => n.id)).toEqual(["m-9"]);
    expect([isStale(client, ["getTicketNotes", "t-1"]), isStale(client, ["getTicketNotes", "t-2"])]).toEqual([false, false]);
  });

  it("refetches a ticket's notes when its thread is first created, since the list had no thread to be patched by", async () => {
    const client = seeded([
      [["getTicketNotes", "t-1"], { conversation_id: "", notes: [] }],
      [["getTicketNotes", "t-2"], { conversation_id: "", notes: [] }],
    ]);
    await followFrame(noteFollower, "chat.conversation.created", { conversation: { id: "c-1", kind: "ticket_thread", ticket_id: "t-1" } }, client);
    expect([isStale(client, ["getTicketNotes", "t-1"]), isStale(client, ["getTicketNotes", "t-2"])]).toEqual([true, false]);
  });
});
