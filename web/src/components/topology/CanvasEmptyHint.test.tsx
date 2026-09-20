import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { CanvasEmptyHint } from "@/components/topology/CanvasEmptyHint";

const renderHint = (onAddNode: () => void) =>
  render(
    <MemoryRouter>
      <CanvasEmptyHint onAddNode={onAddNode} />
    </MemoryRouter>,
  );

describe("CanvasEmptyHint", () => {
  it("renders the empty-canvas invitation copy", () => {
    renderHint(() => {});
    expect(screen.getByText("Your infra, drawn like you'd explain it.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add a node/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /add a service/i })).toHaveAttribute("href", "/wizard/project/repository");
  });

  it("opens the add-node flow from the CTA", () => {
    const onAddNode = vi.fn();
    renderHint(onAddNode);
    fireEvent.click(screen.getByRole("button", { name: /add a node/i }));
    expect(onAddNode).toHaveBeenCalledTimes(1);
  });
});
