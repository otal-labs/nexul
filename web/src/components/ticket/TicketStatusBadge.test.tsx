import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TicketStatusBadge } from "@/components/ticket/TicketStatusBadge";
import { TicketStatus } from "@/models/Ticket";

describe("TicketStatusBadge", () => {
  it("renders each status as a colored icon + text, with no fill/border", () => {
    const { container, rerender } = render(<TicketStatusBadge status={TicketStatus.Open} />);
    let badge = screen.getByText("open");
    expect(badge).toHaveClass("text-info");
    expect(badge).not.toHaveClass("bg-info/15");
    expect(container.querySelector("svg[aria-hidden]")).toBeInTheDocument();

    rerender(<TicketStatusBadge status={TicketStatus.InProgress} />);
    badge = screen.getByText("in progress");
    expect(badge).toHaveClass("text-warning");

    rerender(<TicketStatusBadge status={TicketStatus.Done} />);
    badge = screen.getByText("done");
    expect(badge).toHaveClass("text-success");

    rerender(<TicketStatusBadge status={TicketStatus.Closed} />);
    badge = screen.getByText("closed");
    expect(badge).toHaveClass("text-muted-foreground");
  });
});
