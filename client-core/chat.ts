import type { Embed } from "@nexul/client-core/embed";
import { personLabel, type Person } from "@nexul/client-core/person";

// Mirrors internal/chat/model.go; "voice_channel" is a real conversation that internal/voice attaches a LiveKit room to.
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
  // A DM's people, or a private channel's members.
  participant_ids?: string[];
  // The workspace's own #general: it can be renamed but never deleted, and is always public.
  general?: boolean;
  // Only its members (and the Owner) see and read a private channel (ADR 0098).
  private?: boolean;
}

export type MentionKind = "user" | "agent";

export interface ChatMention {
  kind: MentionKind;
  handle: string;
}

// Who or what posted a message.
export type AuthorKind = "user" | "agent" | "system" | "bot";

// Reaction is one emoji on a message and who reacted with it, earliest first.
export interface Reaction {
  emoji: string;
  user_ids: string[];
}

// A message as the thread route and the message frames carry it; each app adds the hand-offs it renders.
export interface ChatMessage {
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
  reactions?: Reaction[];
  // The harness its author wrote it in, such as T3, when it was relayed from there (ADR 0126).
  via?: string;
  // A bot's name and avatar as the post showed them, kept through the bot's rename or delete (ADR 0129).
  author_name?: string;
  author_avatar_url?: string;
  embeds?: Embed[];
  // Client-only: an optimistic row shown before the server acks the post.
  pending?: boolean;
  // Client-only: the optimistic row's id this server copy confirmed, so the row keeps its identity (and its entrance) on screen.
  client_key?: string | undefined;
}

// Maps conversation id to the caller's unread message count.
export type UnreadCounts = Record<string, number>;

// Adds or removes one person's emoji; applying the same change twice is a no-op, so a push may echo the viewer's own.
export const applyReaction = (reactions: Reaction[] | undefined, emoji: string, userId: string, reacted: boolean): Reaction[] => {
  const list = reactions ?? [];
  const existing = list.find((r) => r.emoji === emoji);
  if (reacted && !existing) return [...list, { emoji, user_ids: [userId] }];
  return list
    .map((r) => {
      if (r.emoji !== emoji || r.user_ids.includes(userId) === reacted) return r;
      return { ...r, user_ids: reacted ? [...r.user_ids, userId] : r.user_ids.filter((id) => id !== userId) };
    })
    .filter((r) => r.user_ids.length > 0);
};

// A server copy also retires the optimistic row it confirms (same author, same text), whichever of the POST reply or the
// push lands first, and takes that row's id as its client_key so the row stays mounted.
export const upsertMessage = <M extends ChatMessage>(list: M[], message: M): M[] => {
  const confirms = (m: M) => m.pending && m.author_id === message.author_id && m.body === message.body;
  const retired = list.find(confirms);
  const kept = list.filter((m) => !confirms(m));
  if (kept.some((m) => m.id === message.id)) return kept.map((m) => (m.id === message.id ? { ...message, client_key: m.client_key } : m));
  return [...kept, { ...message, client_key: retired?.id }];
};

// A note is an Agent message carrying a markdown file (ADR 0108); it is left mid-turn, so it neither ends a turn nor answers one.
export const isNote = (message: ChatMessage): boolean => message.author_kind === "agent" && !!message.attachment_id;

const CONTINUATION_WINDOW_MS = 5 * 60_000;

// The same bot under another name or avatar is a new author on screen, so it starts its own group.
const sameAuthor = (prev: ChatMessage, curr: ChatMessage): boolean => {
  if (prev.author_kind !== curr.author_kind || prev.author_id !== curr.author_id) return false;
  if (curr.author_kind === "bot") return prev.author_name === curr.author_name && prev.author_avatar_url === curr.author_avatar_url;
  return curr.author_kind === "user";
};

// A message continues the previous one's group when it is the same person's or bot's ordinary message, at most five
// minutes after it and on the same day. Agent turns and system lines carry trails and question cards, so they never group.
export const isContinuation = (prev: ChatMessage | undefined, curr: ChatMessage): boolean => {
  if (prev === undefined || prev.deleted_at) return false;
  if (!sameAuthor(prev, curr)) return false;
  const gapMs = Date.parse(curr.created_at) - Date.parse(prev.created_at);
  if (!(gapMs >= 0 && gapMs <= CONTINUATION_WINDOW_MS)) return false;
  return new Date(prev.created_at).toDateString() === new Date(curr.created_at).toDateString();
};

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
    if (dmCtx?.currentUserId && ids.length > 0) return `${personLabel(dmCtx.resolvePerson(dmCtx.currentUserId))} (you)`;
    return "Direct message";
  }
  if (c.kind === "ticket_thread") return "Ticket thread";
  if (c.kind === "doc_thread") return "Doc thread";
  if (c.kind === "interview_thread") return "Interview";
  return "Thread";
};

export const isAttachmentPath = (src: string): boolean => src.startsWith("/api/attachments/");

export type MessageBodySegment =
  | { kind: "text"; text: string }
  | { kind: "image"; src: string; alt: string }
  | { kind: "code"; code: string };

const imageLinePattern = /^!\[([^\]]*)\]\((.+)\)$/;
const fence = "```";
const languageHintPattern = /^[\w+#.-]*$/;

// Discord-style fences: ```code``` on one line, or ```lang through a line ending in ```; unclosed stays text.
const readCodeBlock = (lines: string[], start: number): { code: string; end: number } | null => {
  const opening = lines[start]?.trim() ?? "";
  if (!opening.startsWith(fence)) return null;
  const rest = opening.slice(fence.length);
  if (rest.length > fence.length && rest.endsWith(fence)) return { code: rest.slice(0, -fence.length), end: start };
  const body = languageHintPattern.test(rest) ? [] : [rest];
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i] ?? "";
    if (!line.trimEnd().endsWith(fence)) {
      body.push(line);
      continue;
    }
    const last = line.trimEnd().slice(0, -fence.length);
    if (last.trim() !== "") body.push(last);
    return { code: body.join("\n"), end: i };
  }
  return null;
};

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
  const lines = body.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? "";
    const block = readCodeBlock(lines, i);
    if (block) {
      flush();
      segments.push({ kind: "code", code: block.code });
      i = block.end;
      continue;
    }
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
