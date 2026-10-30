// @vitest-environment node
import { describe, expect, it } from "vitest";

import css from "@/index.css?raw";

const components = import.meta.glob<string>("./components/**/*.tsx", { query: "?raw", import: "default", eager: true });

const firefoxOnly = "@supports not selector(::-webkit-scrollbar)";

// Drops each Firefox-only block, braces balanced, so what is left is what Chromium and Safari read.
const withoutFirefoxBlocks = (source: string): string => {
  let rest = source;
  for (let start = rest.indexOf(firefoxOnly); start !== -1; start = rest.indexOf(firefoxOnly)) {
    let depth = 0;
    let end = rest.indexOf("{", start);
    for (; end < rest.length; end++) {
      if (rest[end] === "{") depth++;
      if (rest[end] === "}" && --depth === 0) break;
    }
    rest = rest.slice(0, start) + rest.slice(end + 1);
  }
  return rest;
};

describe("scrollbars", () => {
  it("draw one token-coloured thumb with no arrow buttons", () => {
    expect(css).toMatch(/::-webkit-scrollbar-button\s*\{\s*display:\s*none;/);
    expect(css).toMatch(/::-webkit-scrollbar-thumb\s*\{[^}]*var\(--scrollbar\)/);
  });

  it("never set a standard scrollbar width or colour where Chromium reads it, which brings back its arrowed bar", () => {
    const chromium = withoutFirefoxBlocks(css);
    expect(chromium).not.toMatch(/scrollbar-color:/);
    expect(chromium).not.toMatch(/scrollbar-width:\s*thin/);
    for (const [path, source] of Object.entries(components)) {
      expect(source, path).not.toMatch(/\[scrollbar-(color|width:thin)/);
    }
  });
});

describe("avatar stacks", () => {
  it("overlap by at most 4px, so each initial shows whole", () => {
    for (const [path, source] of Object.entries(components)) {
      for (const [, step] of source.matchAll(/-space-x-([\d.]+)/g)) {
        expect(Number(step), path).toBeLessThanOrEqual(1);
      }
    }
  });
});
