import { generateText } from "@tiptap/core";
import { MarkdownManager } from "@tiptap/markdown";
import { generateHTML } from "@tiptap/html";

import { upgradeChecklists } from "@/components/doc/legacyChecklist";
import { buildEditorExtensions } from "@/components/doc/mention/editorExtensions";

export { emptyDocJson } from "@/utils/emptyDocJson";

export function isStructuredBody(body: string): boolean {
  try {
    const parsed = JSON.parse(body);
    return (
      parsed !== null &&
      typeof parsed === "object" &&
      ("type" in parsed || Array.isArray(parsed))
    );
  } catch {
    return false;
  }
}

const extensions = buildEditorExtensions();

let markdownManager: MarkdownManager | null = null;

function manager(): MarkdownManager {
  if (!markdownManager) {
    markdownManager = new MarkdownManager({ extensions });
  }
  return markdownManager;
}

// Legacy markdown goes through the editor's own pipeline, so mention refs extract identically either way.
export function parseBodyToJSON(body: string) {
  if (isStructuredBody(body)) {
    return upgradeChecklists(JSON.parse(body));
  }
  return manager().parse(body);
}

// Legacy markdown docs are parsed through the same pipeline the editor uses so the round-trip stays consistent.
export function bodyToHtml(body: string): string {
  const json = parseBodyToJSON(body);
  // The static markup has no node view to refuse a click, so its task checkboxes are disabled.
  return generateHTML(json, extensions).replaceAll('<input type="checkbox"', '<input type="checkbox" disabled');
}

export function bodyToMarkdown(body: string): string {
  if (!isStructuredBody(body)) {
    return body;
  }
  return manager().serialize(parseBodyToJSON(body));
}

export function bodyToPlainText(body: string): string {
  return generateText(parseBodyToJSON(body), extensions, { blockSeparator: " " }).trim();
}
