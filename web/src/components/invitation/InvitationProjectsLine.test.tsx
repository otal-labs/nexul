import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { InvitationProjectsLine } from "@/components/invitation/InvitationProjectsLine";

const projects = Array.from({ length: 11 }, (_, i) => ({ id: `p-${i + 1}`, name: `Project ${i + 1}` }));

describe("InvitationProjectsLine", () => {
  it("folds the names past the eighth behind a count, and opens them on request", async () => {
    const user = userEvent.setup();
    render(<InvitationProjectsLine projects={projects} />);
    expect(screen.getByText("11 projects:")).toBeInTheDocument();
    expect(screen.getByText(/Project 8$/)).toBeInTheDocument();
    expect(screen.queryByText(/Project 9/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "and 3 more" }));

    expect(screen.getByText(/Project 8, Project 9, Project 10, Project 11$/)).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
