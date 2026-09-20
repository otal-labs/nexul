import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FleetStatRow } from "@/components/runner/FleetStatRow";

describe("FleetStatRow", () => {
  it("shows the online, offline, and queued counts", () => {
    render(<FleetStatRow online={3} offline={1} queued={2} />);
    expect(screen.getByText("Online")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("Offline")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("renders a zero state without erroring on an empty fleet", () => {
    render(<FleetStatRow online={0} offline={0} queued={0} />);
    expect(screen.getAllByText("0")).toHaveLength(3);
  });
});
