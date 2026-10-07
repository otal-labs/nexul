import { parseDiscordMarkdown, type InlinePart } from "@/lib/discordMarkdown";

const inline = (text: string): InlinePart[] => {
  const [line] = parseDiscordMarkdown(text);
  return line && line.kind !== "blank" ? line.parts : [];
};

test("a masked http(s) link links with its formatted label", () => {
  expect(inline("[**PR #42**](https://example.com/pr/42)")).toEqual([
    { kind: "link", url: "https://example.com/pr/42", parts: [{ kind: "bold", parts: [{ kind: "text", text: "PR #42" }] }] },
  ]);
});

test.each([
  ["javascript:", "[click me](javascript:alert(document.cookie))"],
  ["data:", "[click me](data:text/html;base64,PHNjcmlwdD4=)"],
  ["a relative path", "[click me](/api/botwebhooks/bw1/token)"],
])("a masked %s link stays as typed text", (_scheme, text) => {
  const parts = inline(text);
  expect(parts.some((part) => part.kind === "link" || part.kind === "url")).toBe(false);
  expect(parts.map((part) => (part.kind === "text" ? part.text : "")).join("")).toContain("click me");
});

test("a bare URL inside a masked link's label is not a second link", () => {
  expect(inline("[see https://example.com/b](https://example.com/a)")).toEqual([
    { kind: "link", url: "https://example.com/a", parts: [{ kind: "text", text: "see " }, { kind: "text", text: "https://example.com/b" }] },
  ]);
});

test("a bare URL links without the sentence's punctuation", () => {
  expect(inline("Deployed to https://example.com/app.")).toEqual([
    { kind: "text", text: "Deployed to " },
    { kind: "url", url: "https://example.com/app" },
    { kind: "text", text: "." },
  ]);
});
