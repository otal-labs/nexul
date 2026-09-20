import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FilterChipRow, type ChipOption } from "@/components/board/FilterChipRow";

const chips: ChipOption[] = [
  { key: "p-1", label: "Backend", active: false, ariaLabel: "Filter by Backend", onToggle: () => {} },
  { key: "p-2", label: "Frontend", active: true, onToggle: () => {} },
];

describe("FilterChipRow", () => {
  it("renders the title and a chip per option", () => {
    render(<FilterChipRow title="Projects" chips={chips} />);
    expect(screen.getByText("Projects")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Filter by Backend" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Frontend" })).toBeInTheDocument();
  });

  it("reflects the active state on aria-pressed", () => {
    render(<FilterChipRow title="Projects" chips={chips} />);
    expect(screen.getByRole("button", { name: "Filter by Backend" })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: "Frontend" })).toHaveAttribute("aria-pressed", "true");
  });

  it("prefers the aria label over the chip text", () => {
    render(<FilterChipRow title="Projects" chips={chips} />);
    expect(screen.getByRole("button", { name: "Filter by Backend" })).toBeInTheDocument();
  });

  it("calls the option's onToggle on click", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<FilterChipRow title="Projects" chips={[{ ...chips[0]!, onToggle }]} />);
    await user.click(screen.getByRole("button", { name: "Filter by Backend" }));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});
