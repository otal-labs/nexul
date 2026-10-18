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
});
