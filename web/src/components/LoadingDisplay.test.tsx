import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LoadingDisplay } from "@/components/LoadingDisplay";

describe("LoadingDisplay", () => {
  it("renders a loading status with the default label, or a custom one", () => {
    const { rerender } = render(<LoadingDisplay />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading");

    rerender(<LoadingDisplay label="Syncing" />);
    expect(screen.getByRole("status")).toHaveTextContent("Syncing");
  });
});
