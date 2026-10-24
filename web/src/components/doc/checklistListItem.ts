import type { MarkdownToken } from "@tiptap/core";
import { ListItem } from "@tiptap/extension-list";

const checkboxAsText = (t: MarkdownToken): MarkdownToken =>
  t.type === "checkbox" ? { type: "text", raw: t.raw ?? "", text: t.raw ?? "" } : t;

// Keeps a task item's "[ ] " as the start of its text, so the item parses like any other.
const withMarkerAsText = (token: MarkdownToken): MarkdownToken => {
  const [first, ...rest] = token.tokens ?? [];
  // A loose item holds the checkbox inside its paragraph.
  if (first?.type === "paragraph") return { ...token, tokens: [{ ...first, tokens: (first.tokens ?? []).map(checkboxAsText) }, ...rest] };
  if (first?.type !== "checkbox") return token;
  // A tight item holds it beside its text, where the stock parser reads the text as a block.
  const marker = checkboxAsText(first);
  const [next, ...after] = rest;
  if (next?.type === "text") return { ...token, tokens: [{ ...next, tokens: [marker, ...(next.tokens ?? [])] }, ...after] };
  return { ...token, tokens: [{ ...marker, tokens: [marker] }, ...rest] };
};

// The editor has no task list node: left to the stock parser, a "- [ ]" line becomes a list item the schema rejects,
// and a live room seeded with it comes back blank and read-only.
export const ChecklistListItem = ListItem.extend({
  parseMarkdown(token, helpers) {
    return this.parent?.(withMarkerAsText(token), helpers) ?? [];
  },
});
