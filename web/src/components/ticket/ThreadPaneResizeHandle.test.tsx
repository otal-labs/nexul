import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ThreadPaneResizeHandle } from "@/components/ticket/ThreadPaneResizeHandle";
import { THREAD_PANE_MAX, useThreadPaneStore } from "@/stores/threadPaneStore";

const separator = () => screen.getByRole("separator", { name: "Resize thread" });
const grid = () => document.querySelector<HTMLElement>("[data-thread-grid]");
const savedWidth = () => useThreadPaneStore.getState().widths.t1;

// jsdom measures nothing; the pane is the handle's parent, so give that element the width the layout would show.
let paneWidth = 0;

const measurePane = () =>
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
    return { width: this === separator().parentElement ? paneWidth : 0 } as DOMRect;
  });

describe("ThreadPaneResizeHandle", () => {
  beforeEach(() => {
    localStorage.clear();
    useThreadPaneStore.setState({ widths: {} });
    paneWidth = 400;
    measurePane();
    render(
      <div data-thread-grid="">
        <div>
          <ThreadPaneResizeHandle ticketId="t1" />
        </div>
      </div>,
    );
  });

  afterEach(() => vi.restoreAllMocks());

  it("steps from the width the pane shows when none is saved yet", async () => {
    const user = userEvent.setup();
    separator().focus();
    await user.keyboard("{ArrowRight}");
    expect(savedWidth()).toBe(416);
  });

  it("paints the dragged width on the ticket grid and saves what the layout shows when it caps the pane", async () => {
    const user = userEvent.setup();
    await user.pointer({ keys: "[MouseLeft>]", target: separator(), coords: { clientX: 400 } });
    await user.pointer({ coords: { clientX: 700 } });
    await new Promise(requestAnimationFrame);
    expect(grid()?.style.getPropertyValue("--thread-pane-width")).toBe(`${THREAD_PANE_MAX}px`);
    paneWidth = 500;
    await user.pointer({ keys: "[/MouseLeft]" });
    expect(savedWidth()).toBe(500);
  });

  it("keeps the pane following the page when the handle is only clicked", async () => {
    const user = userEvent.setup();
    await user.click(separator());
    expect(savedWidth()).toBeUndefined();
  });

  it("goes back to the pane's share of the page on double-click", async () => {
    const user = userEvent.setup();
    separator().focus();
    await user.keyboard("{End}");
    expect(separator()).toHaveAttribute("aria-valuenow", String(THREAD_PANE_MAX));
    await user.dblClick(separator());
    expect(separator()).not.toHaveAttribute("aria-valuenow");
  });
});
