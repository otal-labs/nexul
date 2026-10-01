import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";

import { ListPaneResizeHandle } from "@/components/listpane/ListPaneResizeHandle";
import { LIST_PANE_DEFAULT, LIST_PANE_MAX, LIST_PANE_MIN, useListPaneStore } from "@/stores/listPaneStore";

const width = () => Number(screen.getByRole("separator", { name: "Resize list" }).getAttribute("aria-valuenow"));

describe("ListPaneResizeHandle", () => {
  beforeEach(() => {
    localStorage.clear();
    useListPaneStore.getState().reset();
    render(<ListPaneResizeHandle />);
  });

  it("exposes the width and its bounds on a focusable vertical separator", () => {
    const separator = screen.getByRole("separator", { name: "Resize list" });
    expect(separator).toHaveAttribute("aria-orientation", "vertical");
    expect(separator).toHaveAttribute("aria-valuemin", String(LIST_PANE_MIN));
    expect(separator).toHaveAttribute("aria-valuemax", String(LIST_PANE_MAX));
    expect(width()).toBe(LIST_PANE_DEFAULT);
    expect(separator).toHaveAttribute("tabindex", "0");
  });

  it("steps 16px per arrow key", async () => {
    const user = userEvent.setup();
    screen.getByRole("separator").focus();
    await user.keyboard("{ArrowRight}");
    expect(width()).toBe(LIST_PANE_DEFAULT + 16);
    await user.keyboard("{ArrowLeft}{ArrowLeft}");
    expect(width()).toBe(LIST_PANE_DEFAULT - 16);
  });

  it("stops at the bounds and jumps to them with Home and End", async () => {
    const user = userEvent.setup();
    screen.getByRole("separator").focus();
    await user.keyboard("{End}{ArrowRight}");
    expect(width()).toBe(LIST_PANE_MAX);
    await user.keyboard("{Home}{ArrowLeft}");
    expect(width()).toBe(LIST_PANE_MIN);
  });

  it("resets to the default on double-click", async () => {
    const user = userEvent.setup();
    screen.getByRole("separator").focus();
    await user.keyboard("{End}");
    await user.dblClick(screen.getByRole("separator"));
    expect(width()).toBe(LIST_PANE_DEFAULT);
  });
});
