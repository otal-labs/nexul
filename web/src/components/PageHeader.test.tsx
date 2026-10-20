import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { PageHeader } from "@/components/PageHeader";

class FiringResizeObserver {
  constructor(private callback: ResizeObserverCallback) {}
  observe() {
    this.callback([], this as unknown as ResizeObserver);
  }
  unobserve() {}
  disconnect() {}
}

// Lays the title out as a box of boxHeight holding content of contentHeight, with the rounded scroll heights a page zoom gives.
const layOut = (boxHeight: number, contentHeight: number) => {
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
    return { height: this.classList.contains("peer") ? boxHeight : contentHeight } as DOMRect;
  });
  vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockReturnValue(Math.ceil(contentHeight) + 1);
  vi.spyOn(HTMLElement.prototype, "clientHeight", "get").mockReturnValue(Math.floor(boxHeight));
};

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("ClampedTitle", () => {
  it.each([
    { scenario: "a one-line title at a zoom that rounds its scroll height up is not clamped", box: 39.2, content: 39.2, clamped: false },
    { scenario: "a title a line past the box is clamped", box: 94.08, content: 125.44, clamped: true },
  ])("$scenario", ({ box, content, clamped }) => {
    layOut(box, content);
    vi.stubGlobal("ResizeObserver", FiringResizeObserver);
    render(<PageHeader title="Writing unit and integration tests" />);

    expect(screen.getByRole("heading").parentElement?.parentElement?.hasAttribute("data-clamped")).toBe(clamped);
  });
});
