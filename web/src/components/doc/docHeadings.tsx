import type { JSONContent } from "@tiptap/core";

import { parseBodyToJSON } from "@/utils/RichtextUtility";

export interface DocHeading {
  id: string;
  text: string;
  level: number;
}

// Duplicate heading text gets an index suffix so every row keeps a stable key.
function slugify(text: string): string {
  const slug = text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return slug || "section";
}

// Parses via the editor's own path (JSON or legacy markdown) so headings match the rendered text.
export function extractDocHeadings(body: string): DocHeading[] {
  return extractHeadingsFromJSON(parseBodyToJSON(body));
}

// Exposed separately so a live editor can recompute from its own JSONContent, no reparse round trip.
export function extractHeadingsFromJSON(json: JSONContent | null): DocHeading[] {
  const out: DocHeading[] = [];
  const counts = new Map<string, number>();

  const textOf = (node: JSONContent | null | undefined): string => {
    if (!node) return "";
    if (node.text) return node.text;
    return (node.content ?? []).map(textOf).join("");
  };

  const visit = (node: JSONContent | null | undefined) => {
    if (!node) return;
    if (node.type === "heading" && typeof node.attrs?.level === "number") {
      const text = textOf(node).trim();
      if (!text) return;
      const count = counts.get(text) ?? 0;
      counts.set(text, count + 1);
      out.push({ id: `${slugify(text)}${count > 0 ? `-${count}` : ""}`, text, level: node.attrs.level });
    }
    for (const child of node.content ?? []) visit(child);
  };

  visit(json);
  return out;
}
