export interface DocListItem {
  id: string;
  project_id: string;
  title: string;
  version: number;
  archived: boolean;
  can_open: boolean;
  updated_at: string;
}

export interface Doc {
  id: string;
  project_id: string;
  title: string;
  body: string;
  version: number;
  archived: boolean;
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

// Mentions have no markdown node of their own; a mono chip is the closest the shared renderer's `code` mark gets.
const renderInlineMarkdown = (nodes: RichNode[]): string =>
  nodes
    .map((node) => {
      if (node.type === "hardBreak") return "  \n";
      if (node.type === "mention") return `\`${String(node.attrs?.label ?? "")}\``;
      if (node.type !== "text") return "";
      return (node.marks ?? []).reduce((text, mark) => wrapMark(text, mark), node.text ?? "");
    })
    .join("");

const renderListMarkdown = (list: RichNode, depth: number): string => {
  const ordered = list.type === "orderedList";
  const start = ordered && typeof list.attrs?.start === "number" ? (list.attrs.start as number) : 1;
  const indent = "  ".repeat(depth);
  return (list.content ?? [])
    .map((item, i) => {
      const marker = ordered ? `${start + i}. ` : "- ";
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
  if (node.type === "bulletList" || node.type === "orderedList") return renderListMarkdown(node, depth);
  // Unrecognized node: render its inline text rather than dropping it silently.
  return renderInlineMarkdown(node.content ?? []);
};

// Converts a doc's stored body to markdown for the shared MessageBody/MessageMarkdown renderer; a legacy
// non-JSON body (none in production data today) passes through unchanged, mirroring richtext.ToMarkdown.
export const richBodyToMarkdown = (body: string): string => {
  const blocks = parseRichBody(body);
  if (!blocks) return body;
  return blocks.map((node) => renderBlockMarkdown(node)).join("\n\n");
};
