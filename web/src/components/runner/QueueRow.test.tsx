import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { QueueRow } from "@/components/runner/QueueRow";

describe("QueueRow", () => {
  it("shows the job service, mono id, and a kind token pill", () => {
    render(<QueueRow job={{ id: "d-1", kind: "deploy", service: "web" }} />);
    expect(screen.getByText("d-1")).toBeInTheDocument();
    expect(screen.getByText("web")).toBeInTheDocument();
    expect(screen.getByText("deploy")).toHaveClass("rounded-full");
  });

  it("falls back to the job kind when no service is set", () => {
    render(<QueueRow job={{ id: "d-2", kind: "build" }} />);
    expect(screen.getByText("d-2")).toBeInTheDocument();
    expect(screen.getByText("build")).toBeInTheDocument();
  });

  it("shows a queue position chip when position is provided", () => {
    render(<QueueRow job={{ id: "d-3", kind: "deploy", service: "web" }} position={2} />);
    expect(screen.getByText("#2")).toBeInTheDocument();
  });

  it("omits the position chip when no position is given", () => {
    render(<QueueRow job={{ id: "d-4", kind: "deploy", service: "web" }} />);
    expect(screen.queryByLabelText(/Queue position/)).not.toBeInTheDocument();
  });
});
