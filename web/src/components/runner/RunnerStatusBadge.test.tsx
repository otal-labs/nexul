import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { RunnerStatusBadge } from "@/components/runner/RunnerStatusBadge";

describe("RunnerStatusBadge", () => {
  it("shows a colored wifi icon and success-colored text for a connected runner", () => {
    const { container } = render(<RunnerStatusBadge connected />);
    const badge = screen.getByText("online");
    expect(badge).toHaveClass("text-success");
    expect(badge).not.toHaveClass("bg-success");
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });

  it("shows a muted wifi-off icon and muted text for a disconnected runner", () => {
    const { container } = render(<RunnerStatusBadge connected={false} />);
    const badge = screen.getByText("offline");
    expect(badge).toHaveClass("text-muted-foreground");
    expect(badge).not.toHaveClass("bg-muted");
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });
});
