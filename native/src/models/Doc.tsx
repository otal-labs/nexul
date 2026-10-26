import type { MentionRef, MentionType } from "@nexul/client-core/mention";

export interface DocListItem {
  id: string;
  project_id: string;
  title: string;
  version: number;
  archived: boolean;
  updated_at: string;
}

export interface Doc {
  id: string;
  project_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
  locked: boolean;
  created_at: string;
  updated_at: string;
}

export interface RichMark {
  type: string;
  attrs?: Record<string, unknown>;
}

// Doc.body is canonical Tiptap JSON (ADR 0026): a {"type":"doc",...} object or a bare top-level array of blocks.
export interface RichNode {
  type: string;
  text?: string;
  marks?: RichMark[];
  attrs?: Record<string, unknown>;
  content?: RichNode[];
}

interface RichDoc {
  type: string;
  content?: RichNode[];
}

// Mirrors richtext.UnmarshalJSON; a non-JSON legacy body (none in production data today, per no-backfill policy)
// returns null so the caller falls back to plain text.
export const parseRichBody = (body: string): RichNode[] | null => {
  try {
    const parsed = JSON.parse(body) as unknown;
    if (Array.isArray(parsed)) return parsed as RichNode[];
    if (typeof parsed === "object" && parsed !== null && (parsed as RichDoc).type === "doc") {
      return (parsed as RichDoc).content ?? [];
    }
    return null;
  } catch {
    return null;
  }
};

const wrapMark = (text: string, mark: RichMark): string => {
  if (mark.type === "bold") return `**${text}**`;
  if (mark.type === "italic") return `*${text}*`;
  if (mark.type === "strike") return `~~${text}~~`;
  if (mark.type === "code") return `\`${text}\``;
  if (mark.type === "link") return `[${text}](${String(mark.attrs?.href ?? "")})`;
  return text;
};

// A person mention reads as their name; a doc or ticket one was already written by the body's MentionWriter.
const mentionMarkdown = (attrs: Record<string, unknown> = {}): string => {
  if (attrs.type === "person") return `**@${String(attrs.label ?? "")}**`;
  return "";
};

// Names a person mention by user id; undefined keeps the login the mention was saved with.
export type MentionName = (userId: string) => string | undefined;

// Writes a doc or ticket mention as markdown, from its target and the label it was saved with.
export type MentionWriter = (type: MentionType, id: string, label: string) => string;

// Mirrors richtext.MentionHref: a mention is a link to its target.
export const linkMention: MentionWriter = (type, id, label) => `[${label}](/${type === "ticket" ? "tickets" : "docs"}/${id})`;

const isRecordMention = (node: RichNode): node is RichNode & { attrs: { type: MentionType } } =>
  node.type === "mention" && (node.attrs?.type === "doc" || node.attrs?.type === "ticket");

// Swaps each person mention's stored login for the name People gives it now, and writes each doc and ticket mention.
const writeMentions = (nodes: RichNode[], nameFor: MentionName, write: MentionWriter): RichNode[] =>
  nodes.map((node) => {
    if (isRecordMention(node)) return { type: "text", text: write(node.attrs.type, String(node.attrs.id ?? ""), String(node.attrs.label ?? "")) };
    const name = node.type === "mention" && node.attrs?.type === "person" ? nameFor(String(node.attrs.id ?? "")) : undefined;
    const content = node.content && writeMentions(node.content, nameFor, write);
    return { ...node, ...(name !== undefined && { attrs: { ...node.attrs, label: name } }), ...(content && { content }) };
  });

// Every doc and ticket a body mentions, once each, in the order they appear.
export const mentionRefs = (body: string): MentionRef[] => {
  const refs = new Map<string, MentionRef>();
  const visit = (nodes: RichNode[]) => {
    for (const node of nodes) {
      const id = String(node.attrs?.id ?? "");
      if (isRecordMention(node) && id) refs.set(`${node.attrs.type}:${id}`, { type: node.attrs.type, id });
      visit(node.content ?? []);
    }
  };
  visit(parseRichBody(body) ?? []);
  return [...refs.values()];
};

const renderInlineMarkdown = (nodes: RichNode[]): string =>
  nodes
    .map((node) => {
      if (node.type === "hardBreak") return "  \n";
      if (node.type === "mention") return mentionMarkdown(node.attrs);
      if (node.type !== "text") return "";
      return (node.marks ?? []).reduce((text, mark) => wrapMark(text, mark), node.text ?? "");
    })
    .join("");

const listMarker = (list: RichNode, item: RichNode, index: number, start: number): string => {
  if (list.type === "taskList") return item.attrs?.checked === true ? "- [x] " : "- [ ] ";
  return list.type === "orderedList" ? `${start + index}. ` : "- ";
};

const renderListMarkdown = (list: RichNode, depth: number): string => {
  const ordered = list.type === "orderedList";
  const start = ordered && typeof list.attrs?.start === "number" ? (list.attrs.start as number) : 1;
  const indent = "  ".repeat(depth);
  return (list.content ?? [])
    .map((item, i) => {
      const marker = listMarker(list, item, i, start);
      const [first, ...rest] = item.content ?? [];
      const firstLine = first ? renderBlockMarkdown(first, depth) : "";
      const restLines = rest.map((child) => renderBlockMarkdown(child, depth + 1)).join("\n");
      return `${indent}${marker}${firstLine}${restLines ? `\n${restLines}` : ""}`;
    })
    .join("\n");
};

// One node of the Tiptap tree (ADR 0026), mirroring internal/docs/richtext's renderBlock so the phone reads
// the same content the web editor produces, through the shared markdown renderer rather than a second one.
const renderBlockMarkdown = (node: RichNode, depth = 0): string => {
  if (node.type === "paragraph") return renderInlineMarkdown(node.content ?? []);
  if (node.type === "heading") {
    const level = Math.min(Math.max(Number(node.attrs?.level) || 1, 1), 6);
    return `${"#".repeat(level)} ${renderInlineMarkdown(node.content ?? [])}`;
  }
  if (node.type === "codeBlock") {
    const lang = String(node.attrs?.language ?? "");
    const code = (node.content ?? []).map((c) => c.text ?? "").join("");
    return `\`\`\`${lang}\n${code}\n\`\`\``;
  }
  if (node.type === "image") {
    return `![${String(node.attrs?.alt ?? "")}](${String(node.attrs?.src ?? "")})`;
  }
  if (node.type === "horizontalRule") return "---";
  if (node.type === "blockquote") {
    return (node.content ?? [])
      .map((child) => renderBlockMarkdown(child, depth))
      .join("\n")
      .split("\n")
      .map((line) => `> ${line}`)
      .join("\n");
  }
  if (node.type === "bulletList" || node.type === "orderedList" || node.type === "taskList") return renderListMarkdown(node, depth);
  // Unrecognized node: render its inline text rather than dropping it silently.
  return renderInlineMarkdown(node.content ?? []);
};

// Converts a doc's stored body to markdown for the shared MessageBody/MessageMarkdown renderer; a legacy
// non-JSON body (none in production data today) passes through unchanged, mirroring richtext.ToMarkdown.
export const richBodyToMarkdown = (body: string, nameFor: MentionName = () => undefined, write: MentionWriter = linkMention): string => {
  const blocks = parseRichBody(body);
  if (!blocks) return body;
  return writeMentions(blocks, nameFor, write)
    .map((node) => renderBlockMarkdown(node))
    .join("\n\n");
};
