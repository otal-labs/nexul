import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { DeployStatus } from "@/models/Service";

describe("DeployStatusBadge", () => {
  it.each([
    [DeployStatus.Pending, "text-warning"],
    [DeployStatus.Running, "text-info"],
    [DeployStatus.Healthy, "text-success"],
    [DeployStatus.Failed, "text-destructive"],
  ])("renders %s as a colored icon + text badge, never a filled chip", (status, color) => {
    const { container } = render(<DeployStatusBadge status={status} />);
    const badge = screen.getByText(status);
    expect(badge).toHaveClass(color);
    expect(badge.className).not.toMatch(/\bbg-/);
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });
});
