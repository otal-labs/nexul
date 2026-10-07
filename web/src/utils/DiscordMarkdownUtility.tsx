import { httpUrl } from "@/models/Embed";
import { tokenizeMessageText } from "@/utils/MessageTextUtility";

export type InlinePart =
  | { kind: "text"; text: string }
  | { kind: "mention"; text: string }
  | { kind: "code"; text: string }
  | { kind: "bold"; parts: InlinePart[] }
  | { kind: "italic"; parts: InlinePart[] }
  | { kind: "link"; url: string; parts: InlinePart[] }
  | { kind: "url"; url: string }
  | { kind: "time"; date: Date; style: string | undefined };

export type MarkdownLine =
  | { kind: "text"; parts: InlinePart[] }
  | { kind: "item"; parts: InlinePart[] }
  | { kind: "quote"; parts: InlinePart[] }
  | { kind: "blank" };

// Leftmost match wins and earlier alternatives win a tie, so code and URLs shield their * and _ from emphasis.
const INLINE =
  /`([^`\n]+)`|\[([^\]\n]+)\]\(([^)\s]+)\)|<t:(-?\d{1,13})(?::([tTdDfFR]))?>|(https?:\/\/[^\s<>"]+)|\*\*(.+?)\*\*|\*([^*\s](?:[^*\n]*[^*\s])?)\*|(?<![\p{L}\p{N}])_([^_\s](?:[^_\n]*[^_\s])?)_(?![\p{L}\p{N}])/gu;

const plainParts = (text: string, mentionHandles: string[]): InlinePart[] =>
  tokenizeMessageText(text, mentionHandles).map((part): InlinePart => (part.kind === "link" ? { kind: "url", url: part.url } : part));

interface InlineContext {
  mentionHandles: string[];
  // A link's label may not hold another link: nested anchors are invalid HTML.
  links: boolean;
}

const matchPart = (m: RegExpExecArray, ctx: InlineContext): InlinePart[] => {
  const [raw, code, label, href, unix, style, url, bold, star, underscore] = m;
  if (code !== undefined) return [{ kind: "code", text: code }];
  if (label !== undefined) {
    const safe = ctx.links ? httpUrl(href) : undefined;
    if (!safe) return [{ kind: "text", text: raw }];
    return [{ kind: "link", url: safe, parts: parseInline(label, { ...ctx, links: false }) }];
  }
  if (unix !== undefined) {
    const date = new Date(Number(unix) * 1000);
    if (Number.isNaN(date.getTime())) return [{ kind: "text", text: raw }];
    return [{ kind: "time", date, style }];
  }
  if (url !== undefined) return ctx.links ? plainParts(url, []) : [{ kind: "text", text: url }];
  if (bold !== undefined) return [{ kind: "bold", parts: parseInline(bold, ctx) }];
  return [{ kind: "italic", parts: parseInline(star ?? underscore ?? "", ctx) }];
};

const parseInline = (text: string, ctx: InlineContext): InlinePart[] => {
  const parts: InlinePart[] = [];
  let cursor = 0;
  for (const m of text.matchAll(INLINE)) {
    parts.push(...plainParts(text.slice(cursor, m.index), ctx.mentionHandles));
    parts.push(...matchPart(m, ctx));
    cursor = m.index + m[0].length;
  }
  parts.push(...plainParts(text.slice(cursor), ctx.mentionHandles));
  return parts;
};

const ITEM = /^\s*[-*]\s+(.*)$/;
const QUOTE = /^>\s?(.*)$/;

// The slice of Discord markdown senders use: bold, italics, inline code, masked links, bare URLs, <t:…> times,
// "- " list lines, and "> " quotes. Anything else reads as typed.
export const parseDiscordMarkdown = (text: string, mentionHandles: string[] = []): MarkdownLine[] =>
  text.split("\n").map((line): MarkdownLine => {
    const ctx = { mentionHandles, links: true };
    const item = ITEM.exec(line);
    if (item) return { kind: "item", parts: parseInline(item[1] ?? "", ctx) };
    const quote = QUOTE.exec(line);
    if (quote) return { kind: "quote", parts: parseInline(quote[1] ?? "", ctx) };
    if (line.trim() === "") return { kind: "blank" };
    return { kind: "text", parts: parseInline(line, ctx) };
  });
