import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { DocToc } from "@/components/doc/DocToc";
import type { DocHeading } from "@/components/doc/docHeadings";

const headings: DocHeading[] = [
  { id: "storage", text: "Storage", level: 1 },
  { id: "backup", text: "Backup", level: 2 },
];

const scrollIntoView = vi.fn();
const bodyView = (markup: string) => {
  document.body.insertAdjacentHTML("afterbegin", `<div class="pane"><div class="doc-body-view">${markup}</div></div>`);
};

// jsdom lays nothing out, so each heading is given the top a scroll would leave it at.
const placeHeadings = (tops: Record<string, number>) => {
  for (const el of document.querySelectorAll<HTMLElement>(".doc-body-view h1, .doc-body-view h2")) {
    const top = tops[el.textContent ?? ""] ?? 0;
    el.getBoundingClientRect = () => ({ top }) as DOMRect;
  }
};

const scrollPane = () => document.querySelector(".pane")?.dispatchEvent(new Event("scroll"));

afterEach(() => {
  document.body.innerHTML = "";
  vi.unstubAllGlobals();
  scrollIntoView.mockReset();
});

describe("DocToc", () => {
  it("renders nothing for a single heading, and appears once a second one is added", () => {
    const { container, rerender } = render(<DocToc headings={headings.slice(0, 1)} />);
    expect(container).toBeEmptyDOMElement();

    rerender(<DocToc headings={headings} />);
    expect(screen.getByLabelText("On this page")).toBeInTheDocument();

    rerender(<DocToc headings={headings.slice(0, 1)} />);
    expect(screen.queryByLabelText("On this page")).not.toBeInTheDocument();
  });

  it("renders one link per heading", () => {
    render(<DocToc headings={headings} />);
    expect(screen.getByRole("link", { name: "Storage" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Backup" })).toBeInTheDocument();
  });

  it("marks the first heading current before the reader scrolls", () => {
    render(<DocToc headings={headings} />);
    expect(screen.getByRole("link", { name: "Storage" })).toHaveAttribute("aria-current", "true");
    expect(screen.getByRole("link", { name: "Backup" })).not.toHaveAttribute("aria-current");
  });

  it("scrolls the matching heading element on click", async () => {
    const user = userEvent.setup();
    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    Element.prototype.scrollIntoView = scrollIntoView;

    render(<DocToc headings={headings} />);
    await user.click(screen.getByRole("link", { name: "Backup" }));

    expect(scrollIntoView).toHaveBeenCalledTimes(1);
    const scrolled = scrollIntoView.mock.calls[0]?.[0] as { block?: string };
    expect(scrolled?.block).toBe("start");
    expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true");
  });

  it("marks the last heading above the reading line as the pane scrolls", async () => {
    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    render(<DocToc headings={headings} />);
    placeHeadings({ Storage: -400, Backup: 100 });

    scrollPane();

    await waitFor(() => expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true"));
    expect(screen.getByRole("link", { name: "Storage" })).not.toHaveAttribute("aria-current");
  });

  it("follows headings the editor renders after the outline mounts, as a session sync replaces them", async () => {
    render(<DocToc headings={headings} />);
    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    placeHeadings({ Storage: -400, Backup: 100 });

    scrollPane();

    await waitFor(() => expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true"));
  });

  it("ignores a scroll box that doesn't hold the body, such as the docs list", async () => {
    bodyView("<h1>Storage</h1><h2>Backup</h2>");
    render(<DocToc headings={headings} />);
    placeHeadings({ Storage: -400, Backup: 100 });
    const list = document.createElement("div");
    document.body.appendChild(list);

    list.dispatchEvent(new Event("scroll"));
    await new Promise((resolve) => requestAnimationFrame(resolve));

    expect(screen.getByRole("link", { name: "Storage" })).toHaveAttribute("aria-current", "true");
  });

  it("tolerates a missing body view", async () => {
    const user = userEvent.setup();
    render(<DocToc headings={headings} />);
    await user.click(screen.getByRole("link", { name: "Backup" }));
    expect(screen.getByRole("link", { name: "Backup" })).toHaveAttribute("aria-current", "true");
  });
});
