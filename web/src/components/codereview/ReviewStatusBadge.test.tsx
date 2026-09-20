import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewStatusBadge } from "@/components/codereview/ReviewStatusBadge";
import { ReviewStatus } from "@/models/CodeReview";

describe("ReviewStatusBadge", () => {
  it.each([
    [ReviewStatus.Pending, "pending"],
    [ReviewStatus.Approved, "approved"],
    [ReviewStatus.ChangesRequested, "changes requested"],
    [ReviewStatus.Merged, "merged"],
    [ReviewStatus.Closed, "closed"],
  ] as const)("renders %s", (status, label) => {
    render(<ReviewStatusBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it.each([
    [ReviewStatus.Pending, "text-info"],
    [ReviewStatus.Approved, "text-success"],
    [ReviewStatus.ChangesRequested, "text-warning"],
    [ReviewStatus.Merged, "text-success"],
    [ReviewStatus.Closed, "text-muted-foreground"],
  ] as const)("renders %s as a no-fill icon badge with no tinted background", (status, colorClass) => {
    const { container } = render(<ReviewStatusBadge status={status} />);
    const badge = container.querySelector('[data-slot="no-fill-badge"]');
    expect(badge).toHaveClass(colorClass);
    expect(badge?.className).not.toMatch(/bg-(success|warning|info|destructive|muted)\/?/);
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();
  });
});
