import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketDueRow } from "@/components/ticket/TicketDueRow";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { withOwner } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), patch: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "tt-task",
  title: "Write migrations",
  body: "",
  status: "st-doing",
  position: 0,
  number: 1,
  doc_id: "",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  due_on: "",
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const renderRow = (due_on = "") =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TicketDueRow ticket={{ ...ticket, due_on }} />
    </QueryClientProvider>,
  );

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.get).mockImplementation(withOwner(() => Promise.resolve({ data: [] })));
  vi.mocked(api.patch).mockResolvedValue({ data: ticket });
});

describe("TicketDueRow", () => {
  it("says there is no due date, and shows a set one as a calendar day", async () => {
    const { unmount } = renderRow();
    expect(await screen.findByRole("button", { name: "Due: none" })).toHaveTextContent("No due date");
    unmount();

    renderRow("2026-10-30");
    expect(await screen.findByRole("button", { name: /^Due: .*2026/ })).toHaveTextContent(/30.*Oct.*2026|Oct.*30.*2026/);
  });

  it("saves the picked date through the due-date route when the popover closes", async () => {
    const user = userEvent.setup();
    renderRow();

    await user.click(await screen.findByRole("button", { name: "Due: none" }));
    await user.type(await screen.findByLabelText("Due date"), "2026-10-30");
    expect(api.patch).not.toHaveBeenCalled();
    await user.keyboard("{Escape}");

    expect(api.patch).toHaveBeenCalledTimes(1);
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/due-date", { due_on: "2026-10-30" });
  });

  it("saves on Enter", async () => {
    const user = userEvent.setup();
    renderRow();

    await user.click(await screen.findByRole("button", { name: "Due: none" }));
    await user.type(await screen.findByLabelText("Due date"), "2026-11-02");
    await user.keyboard("{Enter}");

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/due-date", { due_on: "2026-11-02" });
  });

  it("sends nothing when the date was left as it was", async () => {
    const user = userEvent.setup();
    renderRow("2026-10-30");

    await user.click(await screen.findByRole("button", { name: /^Due: / }));
    await user.keyboard("{Escape}");

    expect(api.patch).not.toHaveBeenCalled();
  });

  it("clears the date with the clear button, and offers it only while there is one", async () => {
    const user = userEvent.setup();
    const { unmount } = renderRow("2026-10-30");

    await user.click(await screen.findByRole("button", { name: /^Due: / }));
    await user.click(await screen.findByRole("button", { name: "Clear due date" }));
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/due-date", { due_on: "" });
    unmount();

    renderRow();
    await user.click(await screen.findByRole("button", { name: "Due: none" }));
    await screen.findByLabelText("Due date");
    expect(screen.queryByRole("button", { name: "Clear due date" })).not.toBeInTheDocument();
  });
});
