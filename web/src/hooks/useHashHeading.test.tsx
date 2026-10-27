import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useHashHeading } from "@/hooks/useHashHeading";

const headings = [
  { id: "setup", text: "Setup", level: 2 },
  { id: "recovery", text: "Recovery", level: 2 },
];

const at = (entry: string) => ({ children }: { children: ReactNode }) => <MemoryRouter initialEntries={[entry]}>{children}</MemoryRouter>;

const drawHeading = (text: string) => {
  const body = document.createElement("div");
  body.className = "doc-body-view";
  const el = document.createElement("h2");
  el.textContent = text;
  body.append(el);
  document.body.append(body);
  return el;
};

afterEach(() => {
  document.body.innerHTML = "";
});

describe("useHashHeading", () => {
  it("scrolls to the heading the hash names once the editor draws it", async () => {
    renderHook(() => useHashHeading(headings), { wrapper: at("/docs/d-1#recovery") });
    const scroll = vi.fn();

    await act(async () => {
      drawHeading("Setup").scrollIntoView = vi.fn();
      drawHeading("Recovery").scrollIntoView = scroll;
      await Promise.resolve();
    });

    expect(scroll).toHaveBeenCalledWith({ block: "start" });
  });

  it("scrolls nowhere when the hash names no heading", async () => {
    const el = drawHeading("Recovery");
    el.scrollIntoView = vi.fn();

    renderHook(() => useHashHeading(headings), { wrapper: at("/docs/d-1#gone") });

    expect(el.scrollIntoView).not.toHaveBeenCalled();
  });
});
