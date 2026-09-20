import { describe, expect, it } from "vitest";

import { bodyToHtml, bodyToMarkdown } from "@/utils/RichtextUtility";

describe("bodyToHtml", () => {
  it("renders structured JSON as HTML", () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"SQLite is the spine"}]}]}`;
    const html = bodyToHtml(body);
    expect(html).toContain("<p>");
    expect(html).toContain("SQLite is the spine");
  });

  it("renders legacy markdown as HTML", () => {
    const html = bodyToHtml("# Heading\n\n**bold** text");
    expect(html).toContain("<h1>");
    expect(html).toContain("<strong>");
  });

  it("renders bold and italic marks", () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"bold","marks":[{"type":"bold"}]},{"type":"text","text":" and "},{"type":"text","text":"italic","marks":[{"type":"italic"}]}]}]}`;
    const html = bodyToHtml(body);
    expect(html).toContain("<strong>");
    expect(html).toContain("<em>");
  });

  it("renders empty and malformed bodies without crashing", () => {
    expect(bodyToHtml("")).toBeDefined();
    expect(bodyToHtml("just markdown")).toContain("just markdown");
  });
});

describe("bodyToMarkdown", () => {
  it("converts structured JSON to markdown", () => {
    const body = `{"type":"doc","content":[{"type":"heading","attrs":{"level":1},"content":[{"type":"text","text":"Hi"}]}]}`;
    expect(bodyToMarkdown(body)).toBe("# Hi");
  });

  it("passes through legacy markdown", () => {
    expect(bodyToMarkdown("legacy ## body")).toBe("legacy ## body");
  });

  it("serializes mention nodes as canonical internal links", () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"see "},{"type":"mention","attrs":{"type":"ticket","id":"t-1","label":"Fix the bug"}},{"type":"text","text":" and "},{"type":"mention","attrs":{"type":"doc","id":"d-9","label":"Architecture"}}]}]}`;
    expect(bodyToMarkdown(body)).toBe("see [Fix the bug](/tickets/t-1) and [Architecture](/docs/d-9)");
  });
});

describe("mentions", () => {
  it("parses canonical internal links in markdown into mention chips on render", () => {
    const html = bodyToHtml("see [Fix the bug](/tickets/t-1) now");
    expect(html).toContain('data-mention-type="ticket"');
    expect(html).toContain('data-mention-id="t-1"');
    expect(html).toContain("Fix the bug");
  });

  it("renders structured mention nodes as chip spans", () => {
    const body = `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"type":"doc","id":"d-9","label":"Architecture"}}]}]}`;
    const html = bodyToHtml(body);
    expect(html).toContain('data-mention-type="doc"');
    expect(html).toContain('data-mention-id="d-9"');
  });

  it("leaves external links as plain links", () => {
    const html = bodyToHtml("see [site](https://example.com) please");
    expect(html).not.toContain('data-mention-type="ticket"');
    expect(html).not.toContain('data-mention-type="doc"');
    expect(html).toContain('href="https://example.com"');
  });
});
