import { z } from "zod";

import type { ChatMessage, Conversation, ConversationKind, MentionKind } from "@nexul/client-core/chat";
import type { Person } from "@nexul/client-core/person";

import { attachmentPath, type Attachment } from "@/models/Attachment";
import type { Handoff } from "@/models/Handoff";
import type { PlayType } from "@/models/Play";

export interface Message extends ChatMessage {
  handoffs?: Handoff[];
}

// The chat.conversation.deleted frame: the conversation is gone, so it names what was deleted.
export interface ConversationDeleted {
  conversation_id: string;
  workspace_id: string;
  kind: ConversationKind;
  name: string;
}

// How a channel is named in a sentence: a text channel with its #, a voice channel as it is.
export const channelMention = (c: { kind: ConversationKind; name?: string | undefined }): string =>
  c.kind === "channel" ? `#${c.name ?? ""}` : (c.name ?? "");

// Whether a channel's settings card has anything to show: the private switch, or a private channel's members.
export const channelHasSettingsCard = (c: Conversation, canEditChannels: boolean): boolean =>
  !!c.private || (canEditChannels && !c.general);

// The ticket, doc, or project interview a thread belongs to, as a play target; null for a conversation no play can run on.
export const conversationPlayTarget = (c: Conversation): { type: PlayType; id: string } | null => {
  if (c.kind === "ticket_thread" && c.ticket_id) return { type: "ticket", id: c.ticket_id };
  if (c.kind === "doc_thread" && c.doc_id) return { type: "doc", id: c.doc_id };
  if (c.kind === "interview_thread" && c.project_id) return { type: "interview", id: c.project_id };
  return null;
};

export const SaveChannelFormSchema = z.object({
  name: z.string().trim().min(1, "Channel name is required"),
});
export type SaveChannelFormData = z.infer<typeof SaveChannelFormSchema>;

export const CreateChannelFormSchema = SaveChannelFormSchema.extend({
  private: z.boolean(),
  member_ids: z.array(z.string()),
});
export type CreateChannelFormData = z.infer<typeof CreateChannelFormSchema>;

// Who stays when a channel turns private, or who joins it; the person switching always stays.
export const ChannelPeopleFormSchema = z.object({
  user_ids: z.array(z.string()),
});
export type ChannelPeopleFormData = z.infer<typeof ChannelPeopleFormSchema>;

export const AddChannelPeopleFormSchema = z.object({
  user_ids: z.array(z.string()).min(1, "Pick at least one person"),
});

// The DM between exactly these people, if one already exists.
export const findDM = (conversations: Conversation[], userIds: string[]): Conversation | undefined => {
  const wanted = new Set(userIds);
  return conversations.find(
    (c) => c.kind === "dm" && c.participant_ids?.length === wanted.size && c.participant_ids.every((id) => wanted.has(id)),
  );
};

export const SaveDMFormSchema = z.object({
  participant_ids: z.array(z.string()).min(1, "Pick at least one person"),
});
export type SaveDMFormData = z.infer<typeof SaveDMFormSchema>;

// AgentMentionHandle mirrors internal/chat/model.go's AgentHandle — the fixed, non-user @Agent target.
export const AgentMentionHandle = "Agent";

export interface MentionCandidate {
  kind: MentionKind;
  handle: string;
}

// buildMentionCandidates backs the composer's @ picker: @Agent is always offered, plus every workspace member.
export const buildMentionCandidates = (people: Person[]): MentionCandidate[] => [
  { kind: "agent", handle: AgentMentionHandle },
  ...people.map((p) => ({ kind: "user" as const, handle: p.login })),
];

// Same prefix-match pattern as the docs/tickets @ picker, applied to a single-token handle.
export const matchesMentionPrefix = (candidate: MentionCandidate, query: string): boolean =>
  query === "" || candidate.handle.toLowerCase().startsWith(query.toLowerCase());

export interface MentionTriggerState {
  // start is the index of "@" itself, so the composer can splice the picked handle back in.
  start: number;
  query: string;
}

// Finds an in-progress "@token" run ending at the caret, so the picker opens only while actively typing one.
export const findMentionTrigger = (textBeforeCaret: string): MentionTriggerState | undefined => {
  const at = textBeforeCaret.lastIndexOf("@");
  if (at === -1) return undefined;
  const before = textBeforeCaret[at - 1];
  if (before !== undefined && !/\s/.test(before)) return undefined;
  const token = textBeforeCaret.slice(at + 1);
  if (/\s/.test(token)) return undefined;
  return { start: at, query: token };
};

// The one line an uploaded image occupies in a message body, matching the doc editor's bodyToMarkdown shape.
export const attachmentMarkdown = (attachment: Attachment): string =>
  `![${attachment.name}](${attachmentPath(attachment.id)})`;

export const composeMessageBody = (text: string, attachments: Attachment[]): string =>
  [text.trim(), ...attachments.map(attachmentMarkdown)].filter((line) => line !== "").join("\n");

export interface GroupedConversations {
  channels: Conversation[];
  voiceChannels: Conversation[];
  dms: Conversation[];
  // Doc threads are the one thread kind listed here — reached from the sidebar as well as the doc header.
  docThreads: Conversation[];
}

// Ticket and channel threads aren't listed here — they're reached from their ticket/channel, not browsed directly.
export const groupConversations = (conversations: Conversation[]): GroupedConversations => ({
  channels: conversations.filter((c) => c.kind === "channel"),
  voiceChannels: conversations.filter((c) => c.kind === "voice_channel"),
  dms: conversations.filter((c) => c.kind === "dm"),
  docThreads: conversations.filter((c) => c.kind === "doc_thread"),
});

// What a bare /chat opens: never a voice channel, since opening one there would not join it.
export const defaultConversation = (conversations: Conversation[]): Conversation | undefined => {
  const { channels, dms, docThreads } = groupConversations(conversations);
  return channels[0] ?? dms[0] ?? docThreads[0];
};
