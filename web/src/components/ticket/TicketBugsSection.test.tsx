import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketBugsSection } from "@/components/ticket/TicketBugsSection";
import type { Ticket } from "@/models/Ticket";
import type { LinkedTicket, TicketLinkSet } from "@/models/TicketLink";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "failed"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const bug: LinkedTicket = { id: "t-6", project_id: "p-1", prefix: "BKS", number: 6, title: "books 500s", status: "open", done: false };

const emptySet: TicketLinkSet = { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false };

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "tt-task",
  title: "frontend /books",
  body: "",
  status: "st-progress" as Ticket["status"],
  position: 0,
  number: 1,
  doc_id: "",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  created_at: "2026-09-01T00:00:00Z",
  updated_at: "2026-09-01T00:00:00Z",
  labels: null,
};

const mockApi = (set: TicketLinkSet) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/tickets/t-1/ticket-links") return { data: set };
    if (url === "/api/statuses") {
      return { data: [{ id: "st-progress", name: "Doing", kind: "progress" }, { id: "st-shipped", name: "Shipped", kind: "done" }] };
    }
    if (url === "/api/ticket-types") {
      return { data: [{ id: "tt-task", name: "task", body_template: "" }, { id: "tt-bug", name: "Bug", body_template: "## Steps" }] };
    }
    if (url === "/api/tickets") return { data: [{ id: "t-1", project_id: "p-1", number: 1, title: "frontend /books" }] };
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "BKS", name: "Books" }] };
    return { data: [] };
  });
};

const renderSection = (overrides: Partial<Ticket> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <TicketBugsSection ticket={{ ...ticket, ...overrides }} />
        <ContextAwareConfirmation.ConfirmationRoot />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.resetAllMocks();
});

describe("TicketBugsSection", () => {
  it("lists the bugs found in an open ticket by key", async () => {
    mockApi({ ...emptySet, bugs_found: [bug] });
    renderSection();
    expect(await screen.findByText("Found in this ticket")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /BKS-6/ })).toHaveAttribute("href", "/acme/tickets/BKS-6");
  });

  it("titles a done ticket's bugs as found after done", async () => {
    mockApi({ ...emptySet, bugs_found: [bug] });
    renderSection({ status: "st-shipped" as Ticket["status"] });
    expect(await screen.findByText("Found after done")).toBeInTheDocument();
    expect(screen.queryByText("Found in this ticket")).not.toBeInTheDocument();
  });

  it("reports a bug found in this ticket with the link pre-filled and the bug template", async () => {
    mockApi(emptySet);
    vi.mocked(api.post).mockResolvedValue({ data: { id: "t-9" } });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Report a bug" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByRole("button", { name: /Found in BKS-1/ })).toBeInTheDocument();
    expect(within(dialog).queryByRole("checkbox", { name: "Origin unknown" })).not.toBeInTheDocument();
    await vi.waitFor(() => expect(within(dialog).getByLabelText("Body")).toHaveTextContent("Steps"));
    await user.type(within(dialog).getByRole("textbox", { name: "Title" }), "books 500s");
    await user.click(within(dialog).getByRole("button", { name: "Report bug" }));
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith(
        "/api/tickets",
        expect.objectContaining({ title: "books 500s", type_id: "tt-bug", origin_id: "t-1", project_id: "p-1" }),
      ),
    );
  });
});
