import { describe, expect, it } from "vitest";

import { parseInstanceLink, tokenizeMessageText } from "@/utils/MessageTextUtility";

describe("tokenizeMessageText", () => {
  it.each([
    {
      name: "plain URL",
      text: "see https://example.com/a?b=1#c now",
      want: [
        { kind: "text", text: "see " },
        { kind: "link", url: "https://example.com/a?b=1#c" },
        { kind: "text", text: " now" },
      ],
    },
    {
      name: "trailing sentence punctuation stays text",
      text: "go to http://example.com/x.",
      want: [
        { kind: "text", text: "go to " },
        { kind: "link", url: "http://example.com/x" },
        { kind: "text", text: "." },
      ],
    },
    {
      name: "URL in parentheses drops the unbalanced close",
      text: "(https://example.com/a)",
      want: [
        { kind: "text", text: "(" },
        { kind: "link", url: "https://example.com/a" },
        { kind: "text", text: ")" },
      ],
    },
    {
      name: "balanced parentheses inside a URL are kept",
      text: "https://en.wikipedia.org/wiki/Go_(game)!",
      want: [
        { kind: "link", url: "https://en.wikipedia.org/wiki/Go_(game)" },
        { kind: "text", text: "!" },
      ],
    },
    {
      name: "two URLs",
      text: "https://a.dev and https://b.dev",
      want: [
        { kind: "link", url: "https://a.dev" },
        { kind: "text", text: " and " },
        { kind: "link", url: "https://b.dev" },
      ],
    },
    {
      name: "URL next to a mention",
      text: "@bob you can join here instead: https://nexul.example.com/acme/chat/01a0",
      want: [
        { kind: "mention", text: "@bob" },
        { kind: "text", text: " you can join here instead: " },
        { kind: "link", url: "https://nexul.example.com/acme/chat/01a0" },
      ],
    },
    {
      name: "the domain of an email address is never a mention",
      text: "write to ann@bob.example.com",
      want: [{ kind: "text", text: "write to ann@bob.example.com" }],
    },
    {
      name: "javascript: is never a link",
      text: "click javascript:alert(document.cookie)",
      want: [{ kind: "text", text: "click javascript:alert(document.cookie)" }],
    },
    {
      name: "other schemes are never links",
      text: "ftp://files.example.com and data:text/html,<b>x</b>",
      want: [{ kind: "text", text: "ftp://files.example.com and data:text/html,<b>x</b>" }],
    },
  ])("$name", ({ text, want }) => {
    expect(tokenizeMessageText(text, ["bob"])).toEqual(want);
  });
});

describe("parseInstanceLink", () => {
  const origin = "https://nexul.example.com";

  it("reads the workspace and section of a same-origin URL", () => {
    expect(parseInstanceLink(`${origin}/acme/chat/c-1?x=1#m`, origin)).toEqual({
      path: "/acme/chat/c-1?x=1#m",
      workspace: "acme",
      section: "chat",
      rest: ["c-1"],
    });
  });

  it("treats an unprefixed page as having no workspace", () => {
    expect(parseInstanceLink(`${origin}/settings/devices`, origin)).toMatchObject({ workspace: undefined, section: "settings" });
  });

  it("returns null for another origin", () => {
    expect(parseInstanceLink("https://evil.dev/acme/chat/c-1", origin)).toBeNull();
  });
});
