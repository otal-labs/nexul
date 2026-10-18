import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ErrorDisplay } from "@/components/ErrorDisplay";

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
