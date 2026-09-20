import { describe, expect, it } from "vitest";

import { extractDocHeadings } from "@/components/doc/docHeadings";

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
