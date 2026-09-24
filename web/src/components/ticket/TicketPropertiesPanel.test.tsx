import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketStatus, type Ticket } from "@/models/Ticket";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

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
  labels: ["backend", "migrations"],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const ticketTypes = [
  { id: "ticket-type-task", name: "task", position: 0, color: "", created_at: "", updated_at: "" },
  { id: "ticket-type-bug", name: "bug", position: 1, color: "", created_at: "", updated_at: "" },
];

const allLabels = ["backend", "migrations", "urgent"];

const renderPanel = (props: Partial<ComponentProps<typeof TicketPropertiesPanel>> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const utils = render(
    <QueryClientProvider client={client}>
      <TicketPropertiesPanel ticket={ticket} {...props} />
    </QueryClientProvider>,
  );
  return {
    ...utils,
    rerenderWithTicket: (nextTicket: Ticket) =>
      utils.rerender(
        <QueryClientProvider client={client}>
          <TicketPropertiesPanel ticket={nextTicket} {...props} />
        </QueryClientProvider>,
      ),
  };
};

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.post).mockReset();
  vi.mocked(api.delete).mockReset();
  vi.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
    if (url === "/api/tickets/labels") return Promise.resolve({ data: allLabels });
    if (url === "/api/ticket-types") return Promise.resolve({ data: ticketTypes });
    return Promise.resolve({ data: [] });
  });
});

describe("TicketPropertiesPanel", () => {
  it("renders the status badge as the popover trigger", () => {
    const onTransition = vi.fn();
    renderPanel({ onTransition });
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("in progress")).toBeInTheDocument();
  });

  it("opens the status popover and fires a transition on pick", async () => {
    const user = userEvent.setup();
    const onTransition = vi.fn();
    renderPanel({ onTransition });

    await user.click(screen.getByRole("button", { name: /in progress/i }));
    await user.click(await screen.findByRole("button", { name: "Move to Done" }));
    expect(onTransition).toHaveBeenCalledWith(TicketStatus.Done);
  });

  it("does not render a status popover trigger when no transition handler is wired", () => {
    renderPanel();
    expect(screen.queryByRole("button", { name: /in progress/i })).not.toBeInTheDocument();
  });

  it("renders the developer and tester rows, falling back to No one", () => {
    const { rerenderWithTicket } = renderPanel();
    expect(screen.getByRole("button", { name: "Developer: onik97" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tester: no one" })).toBeInTheDocument();

    rerenderWithTicket({ ...ticket, developer: "", tester: "lena" });
    expect(screen.getByRole("button", { name: "Developer: no one" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tester: lena" })).toBeInTheDocument();
  });

  it("sets the tester from the member picker through the tester route", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticket, tester: "" } });
    renderPanel();

    await user.click(screen.getByRole("button", { name: "Tester: no one" }));
    await user.click(await screen.findByRole("button", { name: "No one" }));
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/tester", { login: "" });
  });

  it("shows a person reporter by login", () => {
    renderPanel();
    expect(screen.getByText("Reporter")).toBeInTheDocument();
    expect(screen.getAllByText("onik97").length).toBeGreaterThan(0);
  });

  it("shows Nexul with the person it filed for, or the automation's name", () => {
    const { rerenderWithTicket } = renderPanel();
    rerenderWithTicket({ ...ticket, reporter: { kind: "user:mcp", login: "lena" } });
    expect(screen.getByText("Nexul")).toBeInTheDocument();
    expect(screen.getByText("for lena")).toBeInTheDocument();

    rerenderWithTicket({ ...ticket, reporter: { kind: "automation", automation_id: "a-1", automation_name: "Triage" } });
    expect(screen.getByText("Nexul")).toBeInTheDocument();
    expect(screen.getByText("Triage")).toBeInTheDocument();
  });

  it("renders labels as chips and shows None when there are no labels", () => {
    const { rerenderWithTicket } = renderPanel();
    expect(screen.getByText("backend")).toBeInTheDocument();
    expect(screen.getByText("migrations")).toBeInTheDocument();

    rerenderWithTicket({ ...ticket, labels: [] });
    expect(screen.getByText("None")).toBeInTheDocument();
  });

  it("does not render an add-label trigger when no handler is wired", () => {
    renderPanel();
    expect(screen.queryByRole("button", { name: "Add label" })).not.toBeInTheDocument();
  });

  it("adds an existing label picked from the search list", async () => {
    const user = userEvent.setup();
    const onAddLabel = vi.fn();
    renderPanel({ onAddLabel });

    await user.click(screen.getByRole("button", { name: "Add label" }));
    await user.type(await screen.findByLabelText("Search labels"), "urg");
    await user.click(await screen.findByRole("button", { name: "urgent" }));
    expect(onAddLabel).toHaveBeenCalledWith("t-1", "urgent");
  });

  it("offers to create a label that doesn't exist yet", async () => {
    const user = userEvent.setup();
    const onAddLabel = vi.fn();
    renderPanel({ onAddLabel });

    await user.click(screen.getByRole("button", { name: "Add label" }));
    await user.type(await screen.findByLabelText("Search labels"), "brand-new");
    await user.click(await screen.findByRole("button", { name: 'Create "brand-new"' }));
    expect(onAddLabel).toHaveBeenCalledWith("t-1", "brand-new");
  });

  it("removes a label via its hover-revealed remove button", async () => {
    const user = userEvent.setup();
    const onRemoveLabel = vi.fn();
    renderPanel({ onAddLabel: vi.fn(), onRemoveLabel });

    await user.click(screen.getByRole("button", { name: "Remove label backend" }));
    expect(onRemoveLabel).toHaveBeenCalledWith("t-1", "backend");
  });

  it("shows the ticket's type name", async () => {
    renderPanel();
    expect(await screen.findByText("task")).toBeInTheDocument();
  });

  it("does not render a type popover trigger when no handler is wired", async () => {
    renderPanel();
    await screen.findByText("task");
    expect(screen.queryByRole("button", { name: /task/i })).not.toBeInTheDocument();
  });

  it("opens the type popover and sets the type on pick", async () => {
    const user = userEvent.setup();
    const onSetType = vi.fn();
    renderPanel({ onSetType });

    await user.click(await screen.findByRole("button", { name: /task/i }));
    await user.click(await screen.findByRole("button", { name: "bug" }));
    expect(onSetType).toHaveBeenCalledWith("t-1", "ticket-type-bug");
  });

  it("renders the Development group with its own microheader", async () => {
    renderPanel();
    expect(await screen.findByText("Development")).toBeInTheDocument();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });
});
