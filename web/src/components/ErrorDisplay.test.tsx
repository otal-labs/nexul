import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ErrorDisplay } from "@/components/ErrorDisplay";

describe("ErrorDisplay", () => {
  it("renders the default title", () => {
    render(<ErrorDisplay />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Something went wrong" })).toBeInTheDocument();
  });

  it("renders a custom title", () => {
    render(<ErrorDisplay title="Login failed" />);
    expect(screen.getByRole("heading", { name: "Login failed" })).toBeInTheDocument();
  });

  it("parses the backend error envelope", () => {
    render(<ErrorDisplay error={{ response: { data: { message: "bad", code: "bad", errors: { name: ["Name is required"] } } } }} />);
    expect(screen.getByText("Name is required")).toBeInTheDocument();
  });

  it("omits the message when no error is given", () => {
    const { container } = render(<ErrorDisplay />);
    expect(container.querySelector("p")).not.toBeInTheDocument();
  });
});
