import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { StatusTransitionButtons } from "@/components/ticket/StatusTransitionButtons";
import { TicketStatus, type Ticket } from "@/models/Ticket";

const baseTicket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "",
  status: TicketStatus.Open,
  position: 0,
  number: 1,
  doc_id: "doc-1",
  assignee: "",
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

describe("StatusTransitionButtons", () => {
  it("shows the allowed transitions from open", () => {
    render(<StatusTransitionButtons ticket={baseTicket} onTransition={() => {}} />);
    expect(screen.getByRole("button", { name: "Move to In progress" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Move to Done" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Move to Closed" })).toBeInTheDocument();
  });

  it("hides transitions not allowed from closed", () => {
    const closed = { ...baseTicket, status: TicketStatus.Closed };
    render(<StatusTransitionButtons ticket={closed} onTransition={() => {}} />);
    expect(screen.getByRole("button", { name: "Move to Open" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Move to Done" })).not.toBeInTheDocument();
  });

  it("fires onTransition with the target status", async () => {
    const user = userEvent.setup();
    const onTransition = vi.fn();
    render(<StatusTransitionButtons ticket={baseTicket} onTransition={onTransition} />);

    await user.click(screen.getByRole("button", { name: "Move to Done" }));
    expect(onTransition).toHaveBeenCalledWith(TicketStatus.Done);
  });
});
