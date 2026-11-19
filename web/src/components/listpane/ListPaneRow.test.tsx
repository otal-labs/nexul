import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ListPaneRow } from "@/components/listpane/ListPaneRow";

const box = (top: number, bottom: number) => ({ top, bottom, left: 0, right: 100, width: 100, height: bottom - top, x: 0, y: top, toJSON: () => ({}) }) as DOMRect;

afterEach(() => {
  vi.restoreAllMocks();
  delete (HTMLElement.prototype as { scrollIntoView?: unknown }).scrollIntoView;
});

describe("ListPaneRow", () => {
  it("lands the list's top edge between rows when bringing the selected row into view leaves one cut", () => {
    let scrollTop = 0;
    vi.spyOn(HTMLElement.prototype, "scrollTop", "get").mockImplementation(() => scrollTop);
    vi.spyOn(HTMLElement.prototype, "scrollTop", "set").mockImplementation((value: number) => (scrollTop = value));
    HTMLElement.prototype.scrollIntoView = () => (scrollTop = 50);
    // The scroller's top edge is at 100; once scrolled, the row above the selected one straddles it, 20px still showing.
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
      if (this.classList.contains("scroll-edge")) return box(100, 400);
      if (this.textContent?.includes("Above")) return box(80, 120);
      return box(120, 160);
    });

    render(
      <MemoryRouter>
        <div className="scroll-edge">
          <ul>
            <ListPaneRow to="/a" title="Above" selected={false} />
            <ListPaneRow to="/b" title="Picked" selected />
          </ul>
        </div>
      </MemoryRouter>,
    );

    expect(scrollTop).toBe(70);
  });
});
