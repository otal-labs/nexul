import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewRow } from "@/components/codereview/ReviewRow";
import type { CodeReview } from "@/models/CodeReview";

const review: CodeReview = {
  id: "r-1",
  pr_number: 42,
  repo: "acme/app",
  status: "approved",
  reviewer: "alice",
  created_at: "2026-08-02T12:00:00Z",
};

describe("ReviewRow", () => {
  it("shows repo, PR number, reviewer, and status", () => {
    render(<ReviewRow review={review} />);
    expect(screen.getByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("by alice")).toBeInTheDocument();
    expect(screen.getByText("approved")).toBeInTheDocument();
  });

  it("omits the reviewer when there is none", () => {
    render(<ReviewRow review={{ ...review, reviewer: "" }} />);
    expect(screen.getByText("acme/app#42")).toBeInTheDocument();
    expect(screen.queryByText(/by /)).not.toBeInTheDocument();
  });
});
