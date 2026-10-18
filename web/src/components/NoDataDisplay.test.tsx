import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NoDataDisplay } from "@/components/NoDataDisplay";

describe("NoDataDisplay", () => {
  it("renders a status region with the default message, or a custom one", () => {
    const { rerender } = render(<NoDataDisplay />);
    expect(screen.getByRole("status")).toHaveTextContent("Nothing here yet");

    rerender(<NoDataDisplay message="No tickets yet" />);
    expect(screen.getByRole("status")).toHaveTextContent("No tickets yet");
  });
});
