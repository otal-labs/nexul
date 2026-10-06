import { richBodyToMarkdown } from "@/models/Doc";

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
