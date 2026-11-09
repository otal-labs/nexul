import { describe, expect, it, vi } from "vitest";

import { HTML_PAGE_MAX_HEIGHT, HTML_PAGE_MIN_HEIGHT, htmlPageDocument, htmlPageThemeCss, readHtmlPageMessage } from "./htmlPage";

const theme = { appearance: "dark" as const, variables: { foreground: "oklch(0.97 0.003 265)" } };
const bootstrapAt = (doc: string) => doc.indexOf('<style id="nexul-html-page">');

describe("htmlPageDocument", () => {
  it("puts the theme first in an existing head, ahead of the page's own styles", () => {
    const doc = htmlPageDocument("<!doctype html><html><head><style>:root{--foreground:red}</style></head><body>Hi</body></html>", theme);
    expect(bootstrapAt(doc)).toBeGreaterThan(doc.indexOf("<head>"));
    expect(bootstrapAt(doc)).toBeLessThan(doc.indexOf(":root{--foreground:red}"));
  });

  it("never anchors on a head tag written inside a script or a comment", () => {
    const doc = htmlPageDocument('<!-- <head> --><script>const s = "<head>";</script><p>Hi</p>', theme);
    expect(doc.startsWith("<!doctype html><html><head>")).toBe(true);
    expect(doc).toContain('<!-- <head> --><script>const s = "<head>";</script><p>Hi</p>');
  });

  it("wraps a fragment in a full document", () => {
    const doc = htmlPageDocument("<p>Hi</p>", theme);
    expect(doc).toMatch(/^<!doctype html><html><head>.*<\/head><body><p>Hi<\/p><\/body><\/html>$/s);
  });

  it("keeps the page's own charset and viewport", () => {
    const doc = htmlPageDocument('<head><meta charset="utf-8"><meta name="viewport" content="width=500"></head>', theme);
    expect(doc.match(/charset/g)).toHaveLength(1);
    expect(doc.match(/name="viewport"/g)).toHaveLength(1);
  });
});

describe("htmlPageThemeCss", () => {
  it("drops a value's attempt to close the rule or the style tag", () => {
    const css = htmlPageThemeCss({ appearance: "light", variables: { brand: "red;}</style><script>alert(1)</script>", "bad name": "x" } });
    expect(css).not.toContain("</style>");
    expect(css).not.toContain("bad name");
    expect(css).toContain("color-scheme:light;--brand:red/stylescriptalert(1)/script;");
  });

  it("escapes a closing tag inside the font faces", () => {
    expect(htmlPageThemeCss({ ...theme, fontFaces: "</style>" })).not.toContain("</style>");
  });
});

describe("readHtmlPageMessage", () => {
  it("clamps a height into the frame's limits", () => {
    expect(readHtmlPageMessage({ source: "nexul-html-page", type: "height", height: 1 })).toEqual({ type: "height", height: HTML_PAGE_MIN_HEIGHT });
    expect(readHtmlPageMessage({ source: "nexul-html-page", type: "height", height: 1e9 })).toEqual({ type: "height", height: HTML_PAGE_MAX_HEIGHT });
  });

  it("reads the phone's string messages", () => {
    expect(readHtmlPageMessage('{"source":"nexul-html-page","type":"link","url":"https://example.com/a"}')).toEqual({ type: "link", url: "https://example.com/a" });
  });

  it("ignores a link that is not a web address, another sender, and junk", () => {
    expect(readHtmlPageMessage({ source: "nexul-html-page", type: "link", url: "javascript:alert(1)" })).toBeUndefined();
    expect(readHtmlPageMessage({ source: "elsewhere", type: "height", height: 100 })).toBeUndefined();
    expect(readHtmlPageMessage("not json")).toBeUndefined();
    expect(readHtmlPageMessage(null)).toBeUndefined();
  });
});

// Runs the page's bootstrap against stand-ins for the browser, so its height reporting can be driven by hand.
const runBootstrap = () => {
  const script = /<script>(.*?)<\/script>/s.exec(htmlPageDocument("<p>Hi</p>", theme))?.[1] ?? "";
  const root = { height: 100, get scrollHeight() { return this.height; }, clientHeight: 0, getBoundingClientRect() { return { height: this.height }; } };
  const observer = { callback: () => {}, observe: vi.fn(), disconnect: vi.fn() };
  class FakeResizeObserver {
    constructor(callback: () => void) { observer.callback = callback; }
    observe = observer.observe;
    disconnect = observer.disconnect;
  }
  const parent = { postMessage: vi.fn() };
  const fakeWindow = { ResizeObserver: FakeResizeObserver, parent, addEventListener: vi.fn() };
  const fakeDocument = { documentElement: root, body: null, addEventListener: vi.fn() };
  new Function("window", "document", "ResizeObserver", "Date", script)(fakeWindow, fakeDocument, FakeResizeObserver, { now: () => 0 });
  return { root, observer, parent };
};

describe("the page's height reports", () => {
  it("reports each new height to the host", () => {
    const { root, observer, parent } = runBootstrap();

    root.height = 240;
    observer.callback();
    observer.callback();

    expect(parent.postMessage).toHaveBeenCalledTimes(1);
    expect(parent.postMessage).toHaveBeenCalledWith({ source: "nexul-html-page", type: "height", height: 240 }, "*");
  });

  it("stops following a page that grows with its own frame", () => {
    const { root, observer, parent } = runBootstrap();

    for (let i = 0; i < 100; i++) {
      root.height += 2;
      observer.callback();
    }

    expect(parent.postMessage).toHaveBeenCalledTimes(30);
    expect(observer.disconnect).toHaveBeenCalled();
  });
});
