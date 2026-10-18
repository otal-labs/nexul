import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusBadge } from "@/components/topology/StatusBadge";
import { ServiceStatus } from "@/models/Topology";

describe("StatusBadge", () => {
  it.each([
    [ServiceStatus.Healthy, "text-success"],
    [ServiceStatus.Running, "text-info"],
    [ServiceStatus.Stopped, "text-muted-foreground"],
    [ServiceStatus.Failed, "text-destructive"],
  ] as const)("renders %s as a no-fill badge in %s", (status, colorClass) => {
    const { container } = render(<StatusBadge status={status} />);
    const badge = screen.getByText(status);
    expect(badge).toHaveClass(colorClass);
    expect(badge).not.toHaveClass(`bg-${status}/15`);
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });
});
