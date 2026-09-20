import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DocToc } from "@/components/doc/DocToc";
import type { DocHeading } from "@/components/doc/docHeadings";

const headings: DocHeading[] = [
  { id: "storage", text: "Storage", level: 1 },
  { id: "backup", text: "Backup", level: 2 },
];

class MockIntersectionObserver {
  static instance: MockIntersectionObserver | null = null;
  callback: IntersectionObserverCallback;
  targets = new Set<Element>();
  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback;
    MockIntersectionObserver.instance = this;
  }
  observe(target: Element) {
    this.targets.add(target);
  }
  unobserve(target: Element) {
    this.targets.delete(target);
  }
  disconnect() {
    MockIntersectionObserver.instance = null;
  }
  intersect(...els: Element[]) {
    this.callback(
      els.map((target) => ({ target, isIntersecting: true }) as IntersectionObserverEntry),
      this as unknown as IntersectionObserver,
    );
  }
}

const scrollIntoView = vi.fn();
const bodyView = (markup: string) => {
  document.body.insertAdjacentHTML("afterbegin", `<div class="doc-body-view">${markup}</div>`);
};

afterEach(() => {
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
  scrollIntoView.mockReset();
});

describe("DocToc", () => {
  it("renders nothing for a doc without headings", () => {
    const { container } = render(<DocToc headings={[]} />);
    expect(screen.queryByLabelText("On this page")).not.toBeInTheDocument();
    expect(container).toBeEmptyDOMElement();
  });

  it("renders one link per heading", () => {
    render(<DocToc headings={headings} />);
    expect(screen.getByRole("link", { name: "Storage" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Backup" })).toBeInTheDocument();
  });

  it("scrolls the matching heading element on click", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    Element.prototype.scrollIntoView = scrollIntoView;

    render(<DocToc headings={headings} />);
    await user.click(screen.getByRole("link", { name: "Backup" }));

    expect(scrollIntoView).toHaveBeenCalledTimes(1);
    const scrolled = scrollIntoView.mock.calls[0]?.[0] as { block?: string };
    expect(scrolled?.block).toBe("start");
    expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true");
  });

  it("marks the section in view with the ember active state", () => {
    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
    bodyView("<h1>Storage</h1><h2>Backup</h2>");

    render(<DocToc headings={headings} />);
    const h2 = document.querySelector("h2");
    expect(h2).not.toBeNull();
    act(() => MockIntersectionObserver.instance?.intersect(h2!));

    expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true");
    expect(screen.getByRole("link", { name: "Storage" })).not.toHaveAttribute("aria-current");
  });

  it("attaches the spy once the body view renders (MutationObserver fallback)", async () => {
    vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
    render(<DocToc headings={headings} />);
    expect(MockIntersectionObserver.instance).toBeNull();

    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    await vi.waitFor(() => expect(MockIntersectionObserver.instance).not.toBeNull());
  });

  it("tolerates a missing body view", async () => {
    const user = userEvent.setup();
    render(<DocToc headings={headings} />);
    await user.click(screen.getByRole("link", { name: "Backup" }));
    expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true");
  });
});
