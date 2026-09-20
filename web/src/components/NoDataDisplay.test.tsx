import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NoDataDisplay } from "@/components/NoDataDisplay";

describe("NoDataDisplay", () => {
  it("renders a status region with the default message", () => {
    render(<NoDataDisplay />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByText("Nothing here yet")).toBeInTheDocument();
  });

  it("renders a custom message", () => {
    render(<NoDataDisplay message="No tickets yet" />);
    expect(screen.getByText("No tickets yet")).toBeInTheDocument();
  });
});
