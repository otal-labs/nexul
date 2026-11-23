import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { OverflowCount } from "@/components/OverflowCount";

describe("OverflowCount", () => {
  it("lists what it hides when pressed, without opening or dragging the row it sits in", async () => {
    const user = userEvent.setup();
    const onRowClick = vi.fn();
    const onRowPress = vi.fn();
    render(
      <div onClick={onRowClick} onMouseDown={onRowPress} onTouchStart={onRowPress}>
        <OverflowCount label="2 more labels" items={["i18n", "ux"]} />
      </div>,
    );

    await user.click(screen.getByRole("button", { name: "2 more labels: i18n, ux" }));

    expect(await screen.findByText("i18n")).toBeInTheDocument();
    expect(screen.getByText("ux")).toBeInTheDocument();
    expect(onRowClick).not.toHaveBeenCalled();
    expect(onRowPress).not.toHaveBeenCalled();
  });
});
