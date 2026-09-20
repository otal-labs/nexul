import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { NotificationItem } from "@/components/notifications/NotificationItem";
import type { Notification } from "@/models/Notification";

const notification: Notification = {
  id: "n1",
  user_id: "u1",
  kind: "doc.created",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: false,
  created_at: "2026-08-12T12:00:00Z",
};

const renderItem = (props: Partial<React.ComponentProps<typeof NotificationItem>> = {}) =>
  render(
    <NotificationItem notification={notification} selected={false} onSelect={vi.fn()} {...props} />,
  );

describe("NotificationItem", () => {
  it("renders the subject title and the kind label", () => {
    renderItem();
    expect(screen.getByText("Spec")).toBeInTheDocument();
    expect(screen.getByText("doc created")).toBeInTheDocument();
  });

  it("calls onSelect when clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderItem({ onSelect });

    await user.click(screen.getByRole("button"));

    expect(onSelect).toHaveBeenCalled();
  });

  it("shows an unread indicator for an unread notification", () => {
    renderItem();
    expect(screen.getByLabelText("Unread")).toBeInTheDocument();
  });

  it("hides the unread indicator for a read notification", () => {
    renderItem({ notification: { ...notification, read: true } });
    expect(screen.queryByLabelText("Unread")).not.toBeInTheDocument();
  });

  it("marks itself as the current row when selected", () => {
    renderItem({ selected: true });
    expect(screen.getByRole("button")).toHaveAttribute("aria-current", "true");
  });

  it("renders the memory updated kind label", () => {
    renderItem({
      notification: {
        ...notification,
        kind: "memory.updated",
        subject_type: "memory",
        subject_id: "mem-1",
        subject_title: "Deploy quirks — v3 by onik",
      },
    });
    expect(screen.getByText("Deploy quirks — v3 by onik")).toBeInTheDocument();
    expect(screen.getByText("memory updated")).toBeInTheDocument();
  });

  it("renders a play run outcome pointing at its ticket", () => {
    renderItem({
      notification: {
        ...notification,
        kind: "play.run_finished",
        subject_type: "ticket",
        subject_id: "t-1",
        subject_title: "Fix with AI failed on NEX-12",
      },
    });
    expect(screen.getByText("Fix with AI failed on NEX-12")).toBeInTheDocument();
    expect(screen.getByText("play run ended")).toBeInTheDocument();
  });
});
