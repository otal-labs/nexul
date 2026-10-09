import type { Embed } from "@nexul/client-core/embed";

import { personLabel, type Person } from "@/models/Person";

// Mirrors internal/chat/model.go; "voice_channel" and the interview and channel threads exist but the phone does not list them.
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
  created_by: string;
  created_at: string;
  updated_at: string;
  participant_ids?: string[];
  // Only its members (and the Owner) see and read a private channel (ADR 0098).
  private?: boolean;
}

export type AuthorKind = "user" | "agent" | "system" | "bot";

export interface ChatMention {
  kind: "user" | "agent";
  handle: string;
}

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
  reactions?: Reaction[];
  // Only an Agent reply that handed work to other agents carries it (ADR 0116).
  handoffs?: Handoff[];
  // The harness its author wrote it in, such as T3, when it was relayed from there (ADR 0126).
  via?: string;
  // What a bot message showed when posted, kept through the bot's rename or delete; never set for people (ADR 0129).
  author_name?: string;
  author_avatar_url?: string;
  embeds?: Embed[];
  // Client-only: the optimistic row shown until the server confirms the post.
  pending?: boolean;
}

// Mirrors internal/harness's Handoff states; left_running is work still running in T3 Code when the reply stopped waiting.
export type HandoffState = "running" | "done" | "failed" | "interrupted" | "left_running";

// Mirrors internal/chat.HandoffStep, one step of the handed-off agent's conversation.
export interface HandoffStep {
  kind: "tool_call" | "tool_result" | "text" | "question" | "other" | "note";
  call_id?: string;
  tool?: string;
  summary: string;
  detail?: string;
  at: string;
}

// Mirrors internal/chat.Handoff: the work one helper agent did for an Agent reply.
export interface Handoff {
  id: string;
  driver: string;
  model: string;
  title: string;
  prompt: string;
  state: HandoffState;
  reply: string;
  steps: HandoffStep[];
}

const HANDOFF_STATE_LABEL: Record<HandoffState, string> = {
  running: "Running",
  done: "Done",
  failed: "Failed",
  interrupted: "Interrupted",
  left_running: "Left running",
};

const HANDOFF_STATE_DOT: Record<HandoffState, string> = {
  running: "bg-warning",
  done: "bg-success",
  failed: "bg-destructive",
  interrupted: "bg-muted-foreground",
  left_running: "bg-info",
};

// The server can be newer than the app, so a state it does not know yet still reads and gets the in-flight dot.
export const handoffStateLabel = (state: HandoffState): string => HANDOFF_STATE_LABEL[state] ?? state;

export const handoffStateDot = (state: HandoffState): string => HANDOFF_STATE_DOT[state] ?? HANDOFF_STATE_DOT.running;

// Reaction is one emoji on a message and who reacted with it, earliest first.
export interface Reaction {
  emoji: string;
  user_ids: string[];
}

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

const CONTINUATION_WINDOW_MS = 5 * 60_000;

// The same bot under another name or avatar is a new author on screen, so it starts its own group.
const sameAuthor = (prev: Message, curr: Message): boolean => {
  if (prev.author_kind !== curr.author_kind || prev.author_id !== curr.author_id) return false;
  if (curr.author_kind === "bot") return prev.author_name === curr.author_name && prev.author_avatar_url === curr.author_avatar_url;
  return curr.author_kind === "user";
};

// Mirrors web's isContinuation: the same person's or bot's ordinary message within five minutes and the same day of
// the previous one shares its header. Agent and system lines never group.
export const isContinuation = (prev: Message | undefined, curr: Message): boolean => {
  if (prev === undefined || prev.deleted_at) return false;
  if (!sameAuthor(prev, curr)) return false;
  const gapMs = Date.parse(curr.created_at) - Date.parse(prev.created_at);
  if (!(gapMs >= 0 && gapMs <= CONTINUATION_WINDOW_MS)) return false;
  return new Date(prev.created_at).toDateString() === new Date(curr.created_at).toDateString();
};

export type UnreadCounts = Record<string, number>;

export interface Workspace {
  id: string;
  name: string;
}

export interface DMLabelContext {
  currentUserId: string | undefined;
  resolvePerson: (userId: string) => Person;
}

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
  return "Thread";
};

export interface ConversationGroup {
  title: string;
  conversations: Conversation[];
}

// Voice needs a call UI the phone does not have, and interview and channel threads are reached from their parent on the web.
export const groupConversations = (conversations: Conversation[]): ConversationGroup[] =>
  [
    { title: "Channels", conversations: conversations.filter((c) => c.kind === "channel") },
    { title: "Direct messages", conversations: conversations.filter((c) => c.kind === "dm") },
    {
      title: "Threads",
      conversations: conversations.filter((c) => c.kind === "ticket_thread" || c.kind === "doc_thread"),
    },
  ].filter((group) => group.conversations.length > 0);

export const isAttachmentPath = (src: string): boolean => src.startsWith("/api/attachments/");

export type MessageBodySegment = { kind: "text"; text: string } | { kind: "image"; src: string; alt: string };

const imageLinePattern = /^!\[([^\]]*)\]\((.+)\)$/;

const trimBlankLines = (lines: string[]): string | null => {
  let start = 0;
  let end = lines.length;
  while (start < end && lines[start]?.trim() === "") start++;
  while (end > start && lines[end - 1]?.trim() === "") end--;
  return end > start ? lines.slice(start, end).join("\n") : null;
};

// Same rule as the web: only a whole line pointing at an attachment becomes an image, so a foreign URL never loads.
export const splitMessageBody = (body: string): MessageBodySegment[] => {
  const segments: MessageBodySegment[] = [];
  let buffer: string[] = [];
  const flush = () => {
    const text = trimBlankLines(buffer);
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

// A note is an Agent message carrying a markdown file (ADR 0108).
export const isNote = (message: Message): boolean => message.author_kind === "agent" && !!message.attachment_id;
