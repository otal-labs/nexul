import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { describe, expect, it } from "vitest";

import { extractDocHeadings, extractHeadingsFromJSON, outlineOfDoc } from "@/components/doc/docHeadings";

describe("extractDocHeadings", () => {
  it("returns an empty outline for a body without headings", () => {
    expect(extractDocHeadings("just prose")).toEqual([]);
  });

  it("extracts levels and text from structured JSON", () => {
    const body = JSON.stringify({
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 1 }, content: [{ type: "text", text: "Storage" }] },
        { type: "paragraph", content: [{ type: "text", text: "prose" }] },
        { type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "Backup" }] },
      ],
    });
    expect(extractDocHeadings(body)).toEqual([
      { id: "storage", text: "Storage", level: 1 },
      { id: "backup", text: "Backup", level: 2 },
    ]);
  });

  it("parses legacy markdown headings through the same pipeline", () => {
    expect(extractDocHeadings("# Title\n\n## Section")).toEqual([
      { id: "title", text: "Title", level: 1 },
      { id: "section", text: "Section", level: 2 },
    ]);
  });

  it("dedupes repeated heading text with a suffix", () => {
    const body = JSON.stringify({
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "Caveats" }] },
        { type: "heading", attrs: { level: 3 }, content: [{ type: "text", text: "Caveats" }] },
      ],
    });
    expect(extractDocHeadings(body).map((h) => h.id)).toEqual(["caveats", "caveats-1"]);
  });

  it("skips empty headings and falls back to a stable slug", () => {
    const body = JSON.stringify({
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 1 }, content: [] },
        { type: "heading", attrs: { level: 2 }, content: [{ type: "text", text: "!!! " }] },
      ],
    });
    expect(extractDocHeadings(body)).toEqual([{ id: "section", text: "!!!", level: 2 }]);
  });
});

describe("outlineOfDoc", () => {
  const editorWith = (content: object[]) => new Editor({ extensions: [StarterKit], content: { type: "doc", content } });
  const h = (level: number, text: string) => ({ type: "heading", attrs: { level }, content: text ? [{ type: "text", text }] : [] });

  it("matches the JSON walk, including repeats, empty headings and headings nested in a quote", () => {
    const editor = editorWith([
      h(1, "Plan"),
      h(2, ""),
      { type: "blockquote", content: [h(2, "Plan")] },
      { type: "paragraph", content: [{ type: "text", text: "Plan" }] },
      { type: "heading", attrs: { level: 3 }, content: [{ type: "text", text: "Bold ", marks: [{ type: "bold" }] }, { type: "text", text: "word" }] },
    ]);
    expect(outlineOfDoc(editor.state.doc).headings).toEqual(extractHeadingsFromJSON(editor.getJSON()));
    expect(outlineOfDoc(editor.state.doc).headings.map((x) => x.id)).toEqual(["plan", "plan-1", "bold-word"]);
  });

  it("answers an unchanged doc with the same outline object and a changed doc with a fresh one", () => {
    const editor = editorWith([h(1, "Plan")]);
    const before = outlineOfDoc(editor.state.doc);
    expect(outlineOfDoc(editor.state.doc)).toBe(before);

    editor.commands.insertContentAt(editor.state.doc.content.size, h(2, "Risks"));
    expect(outlineOfDoc(editor.state.doc).headings.map((x) => x.text)).toEqual(["Plan", "Risks"]);
  });
});
