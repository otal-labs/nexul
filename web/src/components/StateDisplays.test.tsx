import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";

describe("LoadingDisplay", () => {
  it("renders a loading status with the default label, or a custom one", () => {
    const { rerender } = render(<LoadingDisplay />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading");

    rerender(<LoadingDisplay label="Syncing" />);
    expect(screen.getByRole("status")).toHaveTextContent("Syncing");
  });
});

describe("NoDataDisplay", () => {
  it("renders a status region with the default message, or a custom one", () => {
    const { rerender } = render(<NoDataDisplay />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing here yet");

    rerender(<NoDataDisplay message="No tickets yet" />);
    expect(screen.getByRole("status")).toHaveTextContent("No tickets yet");
  });
});

describe("EmptyState", () => {
  it("renders the title with its icon, an optional message and action, and a role when given", () => {
    render(<EmptyState title="Empty" message="Create one to get started" role="status" action={<button type="button">Add</button>} />);
    const empty = screen.getByRole("status");
    expect(empty).toHaveTextContent("Empty");
    expect(empty).toHaveTextContent("Create one to get started");
    expect(empty.querySelector("svg")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add" })).toBeInTheDocument();
  });
});

describe("ErrorDisplay", () => {
  it("renders an alert titled by default or by the caller, with no message without an error", () => {
    const { container, rerender } = render(<ErrorDisplay />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Something went wrong" })).toBeInTheDocument();
    expect(container.querySelector("p")).not.toBeInTheDocument();

    rerender(<ErrorDisplay title="Login failed" />);
    expect(screen.getByRole("heading", { name: "Login failed" })).toBeInTheDocument();
  });

  it("parses the backend error envelope", () => {
    render(<ErrorDisplay error={{ response: { data: { message: "bad", code: "bad", errors: { name: ["Name is required"] } } } }} />);
    expect(screen.getByText("Name is required")).toBeInTheDocument();
  });
});
