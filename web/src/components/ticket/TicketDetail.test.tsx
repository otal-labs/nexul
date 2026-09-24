import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TicketDetail } from "@/components/ticket/TicketDetail";
import type { Project } from "@/models/Project";
import { TicketStatus, type Ticket } from "@/models/Ticket";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn().mockResolvedValue({ data: [] }) },
  errorMessage: vi.fn(() => "error"),
}));

interface RenderOpts {
  project?: Project;
  onSave?: (title: string, body: string) => Promise<void> | void;
}

const renderDetail = (ticket: Ticket, opts: RenderOpts = {}) =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <TicketDetail
          ticket={ticket}
          {...(opts.project ? { project: opts.project } : {})}
          {...(opts.onSave ? { onSave: opts.onSave } : {})}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "Add the migration runner.",
  status: TicketStatus.InProgress,
  position: 0,
  number: 1,
  doc_id: "doc-1",
  developer: "onik97",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const project: Project = {
  id: "p-1",
  name: "Backend",
  prefix: "BE",
  position: 0,
  icon: "",
  tests_location: "",
  created_at: "",
  updated_at: "",
};

describe("TicketDetail", () => {
  it("renders the read-only title and body when no onSave is wired", async () => {
    renderDetail(ticket);
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(await screen.findByText("Add the migration runner.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Ticket title" })).not.toBeInTheDocument();
  });

  it("names the reporter in the meta line, as Nexul for a person when an agent filed it", () => {
    const { unmount } = renderDetail(ticket);
    expect(screen.getByText(/ by onik97 · updated/)).toBeInTheDocument();
    unmount();
    renderDetail({ ...ticket, reporter: { kind: "user:mcp", login: "lena" } });
    expect(screen.getByText(/ by Nexul · for lena · updated/)).toBeInTheDocument();
  });

  it("falls back to the raw ticket id when no project is provided", () => {
    renderDetail(ticket);
    expect(screen.getByText("t-1")).toBeInTheDocument();
  });

  it("renders PREFIX-number instead of the raw id when a project is provided", () => {
    renderDetail(ticket, { project });
    expect(screen.getByText("BE-1")).toBeInTheDocument();
    expect(screen.queryByText("t-1")).not.toBeInTheDocument();
  });

  it("the page is the editor: live title input and rich body, no toggle, no Save button", async () => {
    renderDetail(ticket, { onSave: vi.fn() });
    expect(screen.getByRole("textbox", { name: "Ticket title" })).toHaveValue("Write migrations");
    // The e2e suite finds tickets by heading name — the wrapping <h1> must
    // resolve its accessible name from the input's value.
    expect(screen.getByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(await screen.findByLabelText("Ticket description")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /save/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /cancel/i })).not.toBeInTheDocument();
  });

  it("saves an edited title on blur and shows the saved indicator", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderDetail(ticket, { onSave });

    const title = screen.getByRole("textbox", { name: "Ticket title" });
    await user.clear(title);
    await user.type(title, "Sharper title");
    await user.tab();

    await waitFor(() => expect(onSave).toHaveBeenCalledWith("Sharper title", expect.any(String)));
    expect(await screen.findByText(/· saved/)).toBeInTheDocument();
  });

  it("autosaves without blur once the debounce elapses", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderDetail(ticket, { onSave });

    await user.type(screen.getByRole("textbox", { name: "Ticket title" }), "!");
    await waitFor(() => expect(onSave).toHaveBeenCalledWith("Write migrations!", expect.any(String)), {
      timeout: 3000,
    });
  });

  it("does not autosave an untouched ticket — the editor's initial content application is not an edit", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    renderDetail(ticket, { onSave });
    await screen.findByLabelText("Ticket description");
    await new Promise((resolve) => setTimeout(resolve, 1200));
    expect(onSave).not.toHaveBeenCalled();
  });

  it("never saves an empty title", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderDetail(ticket, { onSave });

    await user.clear(screen.getByRole("textbox", { name: "Ticket title" }));
    await user.tab();

    await new Promise((resolve) => setTimeout(resolve, 1000));
    expect(onSave).not.toHaveBeenCalled();
  });
});
