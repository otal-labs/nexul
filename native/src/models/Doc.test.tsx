import { richBodyToMarkdown, splitHtmlPages } from "@/models/Doc";

describe("richBodyToMarkdown", () => {
  test("converts headings, lists, code blocks, links and mentions", () => {
    const body = JSON.stringify({
      type: "doc",
      content: [
        { type: "heading", attrs: { level: 1 }, content: [{ type: "text", text: "Title" }] },
        {
          type: "bulletList",
          content: [
            { type: "listItem", content: [{ type: "paragraph", content: [{ type: "text", text: "First item" }] }] },
          ],
        },
        { type: "codeBlock", attrs: { language: "go" }, content: [{ type: "text", text: 'fmt.Println("hi")' }] },
        {
          type: "paragraph",
          content: [
            { type: "text", text: "See " },
            { type: "text", text: "the other doc", marks: [{ type: "link", attrs: { href: "/docs/doc-2" } }] },
            { type: "text", text: " " },
            { type: "mention", attrs: { type: "doc", id: "doc-3", label: "Spec" } },
            { type: "text", text: " " },
            { type: "mention", attrs: { type: "ticket", id: "t-4", label: "CHK-4" } },
          ],
        },
      ],
    });

    const markdown = richBodyToMarkdown(body);

    expect(markdown).toContain("# Title");
    expect(markdown).toContain("- First item");
    expect(markdown).toContain('```go\nfmt.Println("hi")\n```');
    expect(markdown).toContain("[the other doc](/docs/doc-2)");
    expect(markdown).toContain("[Spec](/docs/doc-3) [CHK-4](/tickets/t-4)");
  });

  test("keeps a task list's checkboxes as the markdown a person would type", () => {
    const item = (checked: boolean, text: string) => ({
      type: "taskItem",
      attrs: { checked },
      content: [{ type: "paragraph", content: [{ type: "text", text }] }],
    });
    const body = JSON.stringify({ type: "doc", content: [{ type: "taskList", content: [item(false, "a"), item(true, "b")] }] });

    expect(richBodyToMarkdown(body)).toBe("- [ ] a\n- [x] b");
  });

  test("passes a non-JSON legacy body through unchanged", () => {
    expect(richBodyToMarkdown("Just plain text")).toBe("Just plain text");
  });
});

describe("richBodyToMarkdown person mentions", () => {
  const body = JSON.stringify({
    type: "doc",
    content: [
      {
        type: "bulletList",
        content: [
          {
            type: "listItem",
            content: [
              {
                type: "paragraph",
                content: [
                  { type: "text", text: "ask " },
                  { type: "mention", attrs: { type: "person", id: "u-nor", label: "norwooddev" } },
                ],
              },
            ],
          },
        ],
      },
    ],
  });

  test("names the person by what People says now, not the saved id", () => {
    expect(richBodyToMarkdown(body, (id) => (id === "u-nor" ? "Nor Wood" : undefined))).toBe("- ask **@Nor Wood**");
  });

  test("keeps the saved login until People loads", () => {
    expect(richBodyToMarkdown(body)).toBe("- ask **@norwooddev**");
  });
});

describe("splitHtmlPages", () => {
  const paragraph = (text: string) => ({ type: "paragraph", content: [{ type: "text", text }] });
  const page = (code: string) => ({ type: "codeBlock", attrs: { language: "html-page" }, content: [{ type: "text", text: code }] });

  test("passes a legacy plain-text body through as one body", () => {
    expect(splitHtmlPages("just text")).toEqual([{ kind: "body", body: "just text" }]);
  });

  test("lifts each page out between the bodies around it, which still render as markdown", () => {
    const parts = splitHtmlPages(
      JSON.stringify({ type: "doc", content: [paragraph("Before"), page("<p>one</p>"), page("<p>two</p>"), paragraph("After")] }),
    );
    expect(parts.map((p) => p.kind)).toEqual(["body", "page", "page", "body"]);
    expect(parts[1]).toEqual({ kind: "page", code: "<p>one</p>" });
    expect(parts.filter((p) => p.kind === "body").map((p) => richBodyToMarkdown(p.body))).toEqual(["Before", "After"]);
  });

  test("keeps an empty page and any other code block in the body as code", () => {
    const parts = splitHtmlPages(
      JSON.stringify({ type: "doc", content: [page(" "), { type: "codeBlock", attrs: { language: "html" }, content: [{ type: "text", text: "<b>x</b>" }] }] }),
    );
    expect(parts).toHaveLength(1);
    expect(parts[0]?.kind === "body" && richBodyToMarkdown(parts[0].body)).toBe("```html-page\n \n```\n\n```html\n<b>x</b>\n```");
  });
});
