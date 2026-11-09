import { httpUrl } from "@nexul/client-core/embed";

export type MessageTextPart = { kind: "text"; text: string } | { kind: "mention"; text: string } | { kind: "link"; url: string };

const urlPattern = /\bhttps?:\/\/[^\s<>"]+/gi;
const trailingPunctuation = ".,;:!?'\"";

const count = (s: string, ch: string) => s.split(ch).length - 1;

// Sentence punctuation and an unbalanced closing bracket belong to the prose around a URL, not to the URL.
const trimUrl = (raw: string): string => {
  let url = raw;
  for (;;) {
    const last = url.at(-1) ?? "";
    if (trailingPunctuation.includes(last)) {
      url = url.slice(0, -1);
      continue;
    }
    if (last === ")" && count(url, "(") < count(url, ")")) {
      url = url.slice(0, -1);
      continue;
    }
    if (last === "]" && count(url, "[") < count(url, "]")) {
      url = url.slice(0, -1);
      continue;
    }
    return url;
  }
};

const splitMentions = (text: string, mentionHandles: string[]): MessageTextPart[] => {
  if (mentionHandles.length === 0) return [{ kind: "text", text }];
  const escaped = mentionHandles.map((h) => h.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  // An @ inside a word is an email address, not a mention.
  const pattern = new RegExp(`((?<![\\w.])@(?:${escaped.join("|")})\\b)`, "gi");
  return text.split(pattern).map((piece, i): MessageTextPart => ({ kind: i % 2 === 1 ? "mention" : "text", text: piece }));
};

// Only http and https become links, so a javascript: or data: URL in a message stays inert text.
export const tokenizeMessageText = (text: string, mentionHandles: string[]): MessageTextPart[] => {
  const parts: MessageTextPart[] = [];
  let cursor = 0;
  for (const match of text.matchAll(urlPattern)) {
    const url = trimUrl(match[0]);
    if (!httpUrl(url)) continue;
    parts.push(...splitMentions(text.slice(cursor, match.index), mentionHandles));
    parts.push({ kind: "link", url });
    cursor = match.index + url.length;
  }
  parts.push(...splitMentions(text.slice(cursor), mentionHandles));
  return parts.filter((part) => part.kind === "link" || part.text !== "");
};
