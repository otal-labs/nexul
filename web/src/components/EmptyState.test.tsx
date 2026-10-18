import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "@/components/EmptyState";

describe("EmptyState", () => {
  it("renders the title with its icon, an optional message and action, and a role when given", () => {
    render(
      <EmptyState title="Empty" message="Create one to get started" role="status" action={<button type="button">Add</button>} />,
    );
    const empty = screen.getByRole("status");
    expect(empty).toHaveTextContent("Empty");
    expect(empty).toHaveTextContent("Create one to get started");
    expect(empty.querySelector("svg")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });
});
