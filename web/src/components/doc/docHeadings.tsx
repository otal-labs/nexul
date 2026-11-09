import type { JSONContent } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";

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

// Shared by the JSON and live-document walks so both produce the same ids.
function outlineOf(raw: { text: string; level: number }[]): DocHeading[] {
  const counts = new Map<string, number>();
  const out: DocHeading[] = [];
  for (const { text: untrimmed, level } of raw) {
    const text = untrimmed.trim();
    if (!text) continue;
    const count = counts.get(text) ?? 0;
    counts.set(text, count + 1);
    out.push({ id: `${slugify(text)}${count > 0 ? `-${count}` : ""}`, text, level });
  }
  return out;
}

// Exposed separately so a body already parsed to JSONContent skips the reparse round trip.
export function extractHeadingsFromJSON(json: JSONContent | null): DocHeading[] {
  const raw: { text: string; level: number }[] = [];

  const textOf = (node: JSONContent | null | undefined): string => {
    if (!node) return "";
    if (node.text) return node.text;
    return (node.content ?? []).map(textOf).join("");
  };

  const visit = (node: JSONContent | null | undefined) => {
    if (!node) return;
    if (node.type === "heading" && typeof node.attrs?.level === "number") raw.push({ text: textOf(node), level: node.attrs.level });
    for (const child of node.content ?? []) visit(child);
  };

  visit(json);
  return outlineOf(raw);
}

export interface DocOutline {
  headings: DocHeading[];
  key: string;
}

const outlines = new WeakMap<ProseMirrorNode, DocOutline>();

// Read off the immutable doc and cached per node, so a cursor-only transaction reuses the last outline.
export function outlineOfDoc(doc: ProseMirrorNode): DocOutline {
  const cached = outlines.get(doc);
  if (cached) return cached;
  const raw: { text: string; level: number }[] = [];
  doc.descendants((node) => {
    if (node.type.name !== "heading") return !node.isTextblock;
    let text = "";
    node.descendants((child) => {
      if (child.isText) text += child.text ?? "";
    });
    raw.push({ text, level: node.attrs.level as number });
    return false;
  });
  const headings = outlineOf(raw);
  const outline = { headings, key: JSON.stringify(headings) };
  outlines.set(doc, outline);
  return outline;
}
