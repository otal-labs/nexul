import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LoadingDisplay } from "@/components/LoadingDisplay";

describe("LoadingDisplay", () => {
  it("renders a loading status with the default label", () => {
    render(<LoadingDisplay />);
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.getByText("Loading")).toBeInTheDocument();
  });

  it("renders a custom label", () => {
    render(<LoadingDisplay label="Syncing" />);
    expect(screen.getByText("Syncing")).toBeInTheDocument();
  });
});
