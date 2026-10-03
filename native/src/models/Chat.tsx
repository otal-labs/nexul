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

export type AuthorKind = "user" | "agent" | "system";

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
  // Client-only: the optimistic row shown until the server confirms the post.
  pending?: boolean;
}

// Reaction is one emoji on a message and who reacted with it, earliest first.
export interface Reaction {
  emoji: string;
  user_ids: string[];
}

const CONTINUATION_WINDOW_MS = 5 * 60_000;

// Mirrors web's isContinuation: the same person's ordinary message within five minutes and the same day of the
// previous one shares its header. Agent and system lines never group.
export const isContinuation = (prev: Message | undefined, curr: Message): boolean => {
  if (prev === undefined || prev.deleted_at) return false;
  if (prev.author_kind !== "user" || curr.author_kind !== "user" || prev.author_id !== curr.author_id) return false;
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
