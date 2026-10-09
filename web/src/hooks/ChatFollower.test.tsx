import type { Location } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";

import { chatFollower } from "@/hooks/ChatFollower";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import type { Conversation, Message } from "@/models/Chat";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const channel: Conversation = { id: "c-1", workspace_id: "ws-1", kind: "channel", name: "eng", created_by: "u-1", created_at: "", updated_at: "" };

const message = (id: string, extra: Partial<Message> = {}): Message => ({
  id,
  conversation_id: "c-1",
  author_id: "u-2",
  author_kind: "user",
  body: id,
  mentions: null,
  created_at: "",
  updated_at: "",
  ...extra,
});

const chat = () =>
  seeded([
    [["getChatConversations", "ws-1"], [channel]],
    [["getChatConversations", "ws-2"], []],
    [["getChatUnread", "ws-1"], {}],
    [["getChatUnread", "ws-2"], {}],
    [["getChatMessages", "c-1", undefined], [message("m-1")]],
  ]);

const at = (pathname: string): Location => ({ pathname, search: "", hash: "", state: null, key: "default" });

describe("the chat follower", () => {
  beforeEach(() => {
    useAgentStreamStore.setState({ streams: {} });
  });

  it("appends, edits and marks deleted the messages of an open conversation from the frames alone", async () => {
    const client = chat();
    await followFrame(chatFollower, "chat.message.created", { message: message("m-2"), workspace_id: "ws-1" }, client);
    await followFrame(chatFollower, "chat.message.updated", { message: message("m-2", { body: "edited" }) }, client);
    await followFrame(chatFollower, "chat.message.deleted", { conversation_id: "c-1", message_id: "m-1", deleted_at: "2026-09-15T00:00:00Z", workspace_id: "ws-1" }, client);
    expect(client.getQueryData<Message[]>(["getChatMessages", "c-1", undefined])?.map((m) => [m.id, m.body, m.deleted_at])).toEqual([
      ["m-1", "m-1", "2026-09-15T00:00:00Z"],
      ["m-2", "edited", undefined],
    ]);
  });

  const keys = [["getChatUnread", "ws-1"], ["getChatUnread", "ws-2"], ["getChatConversations", "ws-1"], ["getChatConversations", "ws-2"]];

  it("refetches only the unread counts of a message's workspace when its list holds the conversation", async () => {
    const client = chat();
    await followFrame(chatFollower, "chat.message.created", { message: message("m-2"), workspace_id: "ws-1" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([true, false, false, false]);
  });

  it("refetches the list and unread counts of only the message's workspace for a conversation its list does not hold yet", async () => {
    const client = chat();
    await followFrame(chatFollower, "chat.message.created", { message: message("m-9", { conversation_id: "dm-new" }), workspace_id: "ws-2" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([false, true, false, true]);
  });

  it("refetches only the unread counts of a deleted message's workspace", async () => {
    const client = chat();
    await followFrame(chatFollower, "chat.message.deleted", { conversation_id: "c-1", message_id: "m-1", deleted_at: "2026-09-15T00:00:00Z", workspace_id: "ws-1" }, client);
    expect(keys.map((key) => isStale(client, key))).toEqual([true, false, false, false]);
  });

  it("drops a deleted conversation from its list, ends its call, and sends its viewer to the chat home", async () => {
    const client = chat();
    useVoiceCallStore.setState({ activeConversationId: "c-1", status: "connected", room: null });
    const deleted = { conversation_id: "c-1", workspace_id: "ws-1", kind: "channel", name: "eng" };
    const { navigate } = await followFrame(chatFollower, "chat.conversation.deleted", deleted, client, at("/acme/chat/c-1"));
    expect(client.getQueryData(["getChatConversations", "ws-1"])).toEqual([]);
    expect(isStale(client, ["getChatConversations", "ws-1"])).toBe(false);
    expect(useVoiceCallStore.getState().activeConversationId).toBeNull();
    expect(navigate).toHaveBeenCalledWith("/acme/chat", { replace: true });
  });

  it("marks a ticket as having a thread on its card and its page once someone starts one", async () => {
    const client = seeded([
      [["getChatThreadIndicators", "p-1"], { "t-2": true }],
      [["getChatTicketThreadStatus", "t-1"], {}],
      [["getChatConversations", "ws-1"], []],
    ]);
    const thread = { ...channel, id: "c-2", kind: "ticket_thread", ticket_id: "t-1", project_id: "p-1" };
    await followFrame(chatFollower, "chat.conversation.created", { conversation: thread }, client);
    expect(client.getQueryData(["getChatThreadIndicators", "p-1"])).toEqual({ "t-2": true, "t-1": true });
    expect(client.getQueryData(["getChatTicketThreadStatus", "t-1"])).toEqual({ "t-1": true });
    expect(isStale(client, ["getChatThreadIndicators", "p-1"])).toBe(false);
  });

  it("shows a ticket's thread as started once a run on it starts", async () => {
    const client = seeded([[["getChatTicketThreadStatus", "t-9"], {}]]);
    const run = { trail_id: "tr-1", play_id: "pl-1", target_type: "ticket", target_id: "t-9", activity: null, ended_at: null, last_error: "" };
    await followFrame(chatFollower, "play.run", { ...run, state: "running" }, client);
    expect(client.getQueryData(["getChatTicketThreadStatus", "t-9"])).toEqual({});
    await followFrame(chatFollower, "play.run", { ...run, state: "starting" }, client);
    expect(client.getQueryData(["getChatTicketThreadStatus", "t-9"])).toEqual({ "t-9": true });
  });

  describe("an @Agent turn's stream", () => {
    const stream = (payload: object) =>
      followFrame(chatFollower, "chat.agent.stream", { conversation_id: "c-1", message_id: "", text: "", streaming: true, ...payload }, chat());
    const helper = (id: string, state: string) => ({ id, driver: "codex", model: "", title: id, prompt: "Review it", state, reply: "", steps: [] });

    it("keeps the bubble's text and tool activity, keyed by conversation", async () => {
      await stream({ message_id: "s-1", text: "Looking into it", activity: "Read main.go started", activity_kind: "tool_call", activity_tool: "Read" });
      expect(useAgentStreamStore.getState().streams["c-1"]).toMatchObject({
        messageId: "s-1",
        text: "Looking into it",
        activity: "Read main.go started",
        activityKind: "tool_call",
        activityTool: "Read",
      });
    });

    it("merges hand-offs by id, keeping earlier ones across frames that carry none", async () => {
      await stream({ handoff: helper("sa-1", "running") });
      await stream({ handoff: helper("sa-2", "running") });
      await stream({ text: "Waiting on the helpers" });
      await stream({ handoff: helper("sa-1", "done") });
      expect(useAgentStreamStore.getState().streams["c-1"]?.handoffs.map((h) => [h.id, h.state])).toEqual([
        ["sa-1", "done"],
        ["sa-2", "running"],
      ]);
    });

    it("yields to the turn's real message, keeps the hand-offs through a question, and ignores notes and people", async () => {
      const client = chat();
      const created = (extra: Partial<Message>) => followFrame(chatFollower, "chat.message.created", { message: message("m-x", extra) }, client);
      await stream({ text: "Asking first", handoff: helper("sa-1", "running") });

      await created({ author_kind: "user" });
      await created({ author_kind: "agent", attachment_id: "f-1" });
      expect(useAgentStreamStore.getState().streams["c-1"]?.text).toBe("Asking first");

      await created({ author_kind: "agent", body: "```nexul-question\n" + JSON.stringify({ request_id: "r-1", questions: [] }) + "\n```" });
      expect(useAgentStreamStore.getState().streams["c-1"]).toMatchObject({ text: "", streaming: true, handoffs: [helper("sa-1", "running")] });

      await created({ author_kind: "agent", body: "All done." });
      expect(useAgentStreamStore.getState().streams["c-1"]).toBeUndefined();
    });

    it("clears the bubble on the pipeline's empty closing frame", async () => {
      await stream({ text: "almost" });
      await stream({ streaming: false });
      expect(useAgentStreamStore.getState().streams["c-1"]).toBeUndefined();
    });
  });
});
