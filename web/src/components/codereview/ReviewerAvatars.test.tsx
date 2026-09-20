import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ReviewerAvatars } from "@/components/codereview/ReviewerAvatars";

describe("ReviewerAvatars", () => {
  it("renders one focusable avatar per reviewer with an accessible name and their GitHub avatar", () => {
    render(<ReviewerAvatars names={["alice", "bob"]} />);
    expect(screen.getByRole("group", { name: "2 reviewers" })).toBeInTheDocument();
    expect(screen.getByLabelText("alice").querySelector("img")).toHaveAttribute("src", "https://github.com/alice.png");
    expect(screen.getByLabelText("bob").querySelector("img")).toHaveAttribute("src", "https://github.com/bob.png");
  });

  it("uses singular wording for one reviewer", () => {
    render(<ReviewerAvatars names={["alice"]} />);
    expect(screen.getByRole("group", { name: "1 reviewer" })).toBeInTheDocument();
  });

  it("renders nothing when every name is empty", () => {
    render(<ReviewerAvatars names={["", "  "] as string[]} />);
    expect(screen.queryByRole("group")).not.toBeInTheDocument();
  });
});
