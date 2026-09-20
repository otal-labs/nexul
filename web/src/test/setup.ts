import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterEach } from "vitest";

// Whole-app imports and lazy dialogs can outlive the 1s default under load; 3s keeps genuine hangs visible.
configure({ asyncUtilTimeout: 3000 });

// Radix portals leave scroll locks and aria-hidden marks when unmounted mid-open; scrub after every test.
afterEach(() => {
  document.body.style.pointerEvents = "";
  document.body.removeAttribute("data-scroll-locked");
  for (const el of document.querySelectorAll("[data-aria-hidden]")) {
    el.removeAttribute("aria-hidden");
    el.removeAttribute("data-aria-hidden");
  }
});

class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}

if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver;
}

// jsdom has no WebSocket; stub it so signed-in tests don't dial the real API origin.
class WebSocketStub {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  readyState: number = WebSocketStub.CONNECTING;
  url: string;
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: unknown) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  constructor(url: string) {
    this.url = url;
  }
  close() {}
  send() {}
}

if (typeof globalThis.WebSocket === "undefined") {
  globalThis.WebSocket = WebSocketStub as unknown as typeof WebSocket;
}

// ProseMirror relies on client rects jsdom doesn't implement; stub them so the editor can mount in tests.
const zeroRect = (): DOMRect => ({
  x: 0,
  y: 0,
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  width: 0,
  height: 0,
  toJSON: () => ({}),
}) as DOMRect;

const zeroRects = (): DOMRectList => ({
  length: 0,
  item: () => null,
  [Symbol.iterator]: [][Symbol.iterator],
}) as DOMRectList;

if (typeof Range !== "undefined") {
  Range.prototype.getBoundingClientRect ??= () => zeroRect();
  Range.prototype.getClientRects ??= () => zeroRects();
}
if (typeof Element !== "undefined") {
  Element.prototype.getBoundingClientRect ??= () => zeroRect();
  Element.prototype.getClientRects ??= () => zeroRects();
}
if (typeof document !== "undefined" && document.createRange) {
  document.createRange = () => {
    const range = new Range();
    return range;
  };
}

// ProseMirror resolves pointer coordinates via the document; jsdom lacks these.
if (typeof document !== "undefined") {
  document.elementFromPoint ??= () => null;
  document.caretRangeFromPoint ??= () => null;
  document.caretPositionFromPoint ??= () => null;
}

afterEach(() => {
  cleanup();
});

// Radix Select needs pointer-capture APIs and scrollIntoView, neither of which jsdom implements.
if (typeof Element.prototype.hasPointerCapture === "undefined") {
  Element.prototype.hasPointerCapture = () => false;
  Element.prototype.setPointerCapture = () => {};
  Element.prototype.releasePointerCapture = () => {};
}
if (typeof Element.prototype.scrollIntoView === "undefined") {
  Element.prototype.scrollIntoView = () => {};
}
