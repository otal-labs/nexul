import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContainerStatusBadge } from "@/components/stack/ContainerStatusBadge";
import { ContainerStatus } from "@/models/Stack";

describe("ContainerStatusBadge", () => {
  it("draws a running container as a still mark, since running is a steady state", () => {
    render(<ContainerStatusBadge status={ContainerStatus.Running} />);
    const icon = screen.getByText(ContainerStatus.Running).querySelector("svg");
    expect(icon).not.toHaveClass("animate-spin");
  });

  it("shows a status the app does not know as its word instead of failing the page", () => {
    render(<ContainerStatusBadge status={"failed" as ContainerStatus} />);
    expect(screen.getByText("failed")).toBeInTheDocument();
  });
});
