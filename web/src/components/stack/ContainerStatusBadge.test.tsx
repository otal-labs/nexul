import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContainerStatusBadge } from "@/components/stack/ContainerStatusBadge";
import { ContainerStatus } from "@/models/Stack";

describe("ContainerStatusBadge", () => {
  it("spins the icon of a running container and stills it under reduced motion", () => {
    render(<ContainerStatusBadge status={ContainerStatus.Running} />);
    const icon = screen.getByText(ContainerStatus.Running).querySelector("svg");
    expect(icon).toHaveClass("animate-spin", "motion-reduce:animate-none");
  });

  it("keeps a healthy container's icon still", () => {
    render(<ContainerStatusBadge status={ContainerStatus.Healthy} />);
    const icon = screen.getByText(ContainerStatus.Healthy).querySelector("svg");
    expect(icon).not.toHaveClass("animate-spin");
  });
});
