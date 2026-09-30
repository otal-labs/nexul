import { z } from "zod";

import { attachmentPath, isAttachmentPath, type Attachment } from "@/models/Attachment";
import { personLabel, type Person } from "@/models/Person";
import type { PlayType } from "@/models/Play";

// "voice_channel" is a real conversation that internal/voice attaches a LiveKit room to.
export type ConversationKind =
  | "channel"
  | "dm"
  | "channel_thread"
  | "ticket_thread"
  | "doc_thread"
  | "interview_thread"
  | "voice_channel";

export interface Conversation {
  id: string;
  workspace_id: string;
  kind: ConversationKind;
  name?: string;
  ticket_id?: string;
  doc_id?: string;
  // Only set on an interview thread.
  project_id?: string;
  parent_message_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  // Only populated for DMs.
  participant_ids?: string[];
  // The workspace's own #general: it can be renamed but never deleted.
  general?: boolean;
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

// The ticket, doc, or project interview a thread belongs to, as a play target; null for a conversation no play can run on.
export const conversationPlayTarget = (c: Conversation): { type: PlayType; id: string } | null => {
  if (c.kind === "ticket_thread" && c.ticket_id) return { type: "ticket", id: c.ticket_id };
  if (c.kind === "doc_thread" && c.doc_id) return { type: "doc", id: c.doc_id };
  if (c.kind === "interview_thread" && c.project_id) return { type: "interview", id: c.project_id };
  return null;
};

export type MentionKind = "user" | "agent";

export interface ChatMention {
  kind: MentionKind;
  handle: string;
}

// Who/what posted a message, for MessageRow's rendering.
export type AuthorKind = "user" | "agent" | "system";

export interface Message {
  id: string;
  conversation_id: string;
  author_id: string;
  author_kind: AuthorKind;
  body: string;
  mentions: ChatMention[] | null;
  attachment_id?: string;
  edited_at?: string;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
  // Client-only: an optimistic row shown before the server acks the post.
  pending?: boolean;
}

const CONTINUATION_WINDOW_MS = 5 * 60_000;

// A message continues the previous one's group when it is the same person's ordinary message, at most five minutes after
// it and on the same day. Agent turns and system lines carry trails and question cards, so they never group.
export const isContinuation = (prev: Message | undefined, curr: Message): boolean => {
  if (prev === undefined || prev.deleted_at) return false;
  if (prev.author_kind !== "user" || curr.author_kind !== "user" || prev.author_id !== curr.author_id) return false;
  const gapMs = Date.parse(curr.created_at) - Date.parse(prev.created_at);
  if (!(gapMs >= 0 && gapMs <= CONTINUATION_WINDOW_MS)) return false;
  return new Date(prev.created_at).toDateString() === new Date(curr.created_at).toDateString();
};

// Maps conversation id to the caller's unread message count.
export type UnreadCounts = Record<string, number>;

export interface DMLabelContext {
  currentUserId: string | undefined;
  resolvePerson: (userId: string) => Person;
}

// Channels use their name; everything else falls back to its kind. Passing dmCtx names a DM by its people.
export const conversationLabel = (c: Conversation, dmCtx?: DMLabelContext): string => {
  if (c.name) return c.name;
  if (c.kind === "dm") {
    const ids = c.participant_ids ?? [];
    const others = ids.filter((id) => id !== dmCtx?.currentUserId);
    if (dmCtx && others.length > 0) return others.map((id) => personLabel(dmCtx.resolvePerson(id))).join(", ");
    if (dmCtx?.currentUserId && ids.length > 0 && others.length === 0) return `${personLabel(dmCtx.resolvePerson(dmCtx.currentUserId))} (you)`;
    return "Direct message";
  }
  if (c.kind === "ticket_thread") return "Ticket thread";
  if (c.kind === "doc_thread") return "Doc thread";
  if (c.kind === "interview_thread") return "Interview";
  return "Thread";
};

export const SaveChannelFormSchema = z.object({
  name: z.string().trim().min(1, "Channel name is required"),
});
export type SaveChannelFormData = z.infer<typeof SaveChannelFormSchema>;

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

export type MessageBodySegment = { kind: "text"; text: string } | { kind: "image"; src: string; alt: string };

const imageLinePattern = /^!\[([^\]]*)\]\((.+)\)$/;

const flushTextLines = (lines: string[]): string | null => {
  let start = 0;
  let end = lines.length;
  while (start < end && lines[start]?.trim() === "") start++;
  while (end > start && lines[end - 1]?.trim() === "") end--;
  return end > start ? lines.slice(start, end).join("\n") : null;
};

// Only whole lines pointing at an attachment path become images, so a foreign image URL never loads.
export const splitMessageBody = (body: string): MessageBodySegment[] => {
  const segments: MessageBodySegment[] = [];
  let buffer: string[] = [];
  const flush = () => {
    const text = flushTextLines(buffer);
    if (text !== null) segments.push({ kind: "text", text });
    buffer = [];
  };
  for (const line of body.split("\n")) {
    const match = line.match(imageLinePattern);
    if (match?.[2] && isAttachmentPath(match[2])) {
      flush();
      segments.push({ kind: "image", src: match[2], alt: match[1] ?? "" });
      continue;
    }
    buffer.push(line);
  }
  flush();
  return segments;
};

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
