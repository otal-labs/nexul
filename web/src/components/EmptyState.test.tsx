import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "@/components/EmptyState";

describe("EmptyState", () => {
  it("renders the title and default icon", () => {
    render(<EmptyState title="No projects" />);
    expect(screen.getByText("No projects")).toBeInTheDocument();
    expect(document.querySelector("svg")).toBeInTheDocument();
  });

  it("renders an optional message", () => {
    render(<EmptyState title="Empty" message="Create one to get started" />);
    expect(screen.getByText("Create one to get started")).toBeInTheDocument();
  });

  it("renders the action slot", () => {
    render(
      <EmptyState title="Empty" action={<button type="button">Add</button>} />,
    );
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });

  it("applies a role when provided", () => {
    render(<EmptyState title="Empty" role="status" />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });
});
