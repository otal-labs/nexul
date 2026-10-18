import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FleetStatRow } from "@/components/runner/FleetStatRow";

describe("FleetStatRow", () => {
  it("shows the online, busy, offline, and queued counts", () => {
    render(<FleetStatRow online={3} busy={4} offline={1} queued={2} />);
    expect(screen.getByText("Online")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText("Offline")).toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
    expect(screen.getByText("Queued")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("Busy")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("of 4")).toBeInTheDocument();
  });
});
