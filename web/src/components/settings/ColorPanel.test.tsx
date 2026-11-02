import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ColorPanel } from "@/components/settings/ColorPanel";

const renderPanel = (value = "") => {
  const onChange = vi.fn();
  const onPick = vi.fn();
  render(<ColorPanel value={value} onChange={onChange} onPick={onPick} />);
  return { onChange, onPick };
};

describe("ColorPanel", () => {
  it("is one tab stop for the quick colors, previews with the arrows and picks with Enter", async () => {
    const user = userEvent.setup();
    const { onChange, onPick } = renderPanel();

    await user.click(screen.getByRole("textbox", { name: "Hex color" }));
    await user.tab();
    expect(screen.getByRole("radio", { name: "No color" })).toHaveFocus();

    // Radix checks the item the arrow lands on only while the key is down, as it is in a browser by the time focus moves.
    await user.keyboard("{ArrowRight>}{/ArrowRight}");
    expect(screen.getByRole("radio", { name: "Red" })).toHaveFocus();
    expect(onChange).toHaveBeenLastCalledWith("#b96558");
    expect(onPick).not.toHaveBeenCalled();

    await user.keyboard("{Enter}");
    expect(onPick).toHaveBeenCalledWith("#b96558");

    await user.tab();
    expect(screen.getByRole("radiogroup", { name: "Quick colors" })).not.toContainElement(document.activeElement as HTMLElement);
  });

  it("picks a quick color on click", async () => {
    const user = userEvent.setup();
    const { onPick } = renderPanel("#b96558");

    expect(screen.getByRole("radio", { name: "Red" })).toBeChecked();
    await user.click(screen.getByRole("radio", { name: "No color" }));

    expect(onPick).toHaveBeenCalledWith("");
  });

  it("flags an invalid hex and puts back the last valid color on blur", async () => {
    const user = userEvent.setup();
    const { onChange } = renderPanel("#3788bb");
    const field = screen.getByRole("textbox", { name: "Hex color" });

    await user.clear(field);
    await user.type(field, "#12zz");
    expect(field).toHaveAttribute("aria-invalid", "true");
    expect(screen.getByRole("alert")).toHaveTextContent("six hex digits");
    expect(onChange).not.toHaveBeenCalled();

    await user.tab();
    expect(field).toHaveValue("#3788bb");
    expect(field).toHaveAttribute("aria-invalid", "false");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
