import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketPage } from "@/pages/TicketPage";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderPage = (entry = "/tickets/t-1") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/tickets/:ticketId" element={<TicketPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const ticketData = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "Add the runner.",
  status: "in_progress",
  number: 7,
  doc_id: "doc-1",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const projectData = [
  { id: "p-1", name: "Backend", prefix: "BE", position: 0, created_at: "", updated_at: "" },
  { id: "p-2", name: "Frontend", prefix: "FE", position: 1, created_at: "", updated_at: "" },
];

const reviewData = [
  {
    id: "r-1",
    pr_number: 42,
    repo: "acme/app",
    ticket_id: "t-1",
    doc_id: "",
    status: "approved",
    reviewer: "alice",
    created_at: "2026-08-02T12:00:00Z",
  },
];

const mockTicket = () => vi.mocked(api.get).mockImplementation((url: string) => {
  if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
  if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
  if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
  if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
  return Promise.resolve({ data: [] });
});

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.patch).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("TicketPage", () => {
  it("renders the ticket with its PREFIX-number id, status control, and reviews", async () => {
    const user = userEvent.setup();
    mockTicket();
    renderPage();
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.getByText("BE-7")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "in progress" }));
    expect(screen.getByRole("button", { name: "Move to Done" })).toBeInTheDocument();
    expect(await screen.findByText("No branches or PRs linked yet.")).toBeInTheDocument();
  });

  it("shows linked pull requests and their review status", async () => {
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
      if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
      if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
      if (url === "/api/reviews") return Promise.resolve({ data: reviewData });
      if (url === "/api/tickets/labels") return Promise.resolve({ data: [] });
      if (url === "/api/ticket-types") return Promise.resolve({ data: [] });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    expect(await screen.findByText("acme/app#42")).toBeInTheDocument();
    expect(screen.getByText("approved")).toBeInTheDocument();
    expect(screen.getByText("by alice")).toBeInTheDocument();
  });

  it("transitions the ticket status", async () => {
    const user = userEvent.setup();
    mockTicket();
    vi.mocked(api.patch).mockResolvedValue({ data: { ...ticketData, status: "done" } });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "in progress" }));
    await user.click(await screen.findByRole("button", { name: "Move to Done" }));
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "done" });
  });

  it("resolves a /tickets/PREFIX-NUMBER url to the same ticket as its uuid url", async () => {
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/tickets") return Promise.resolve({ data: [ticketData] });
      if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
      if (url === "/api/tickets/t-1/links") return Promise.resolve({ data: { prs: [], branches: [] } });
      if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
      if (url.startsWith("/api/projects")) return Promise.resolve({ data: projectData });
      return Promise.resolve({ data: [] });
    });
    renderPage("/tickets/BE-7");
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    expect(screen.getByText("BE-7")).toBeInTheDocument();
  });

  it("shows not-found for a key that matches no ticket", async () => {
    mockTicket();
    renderPage("/tickets/BE-999");
    expect(await screen.findByText("Ticket not found")).toBeInTheDocument();
  });

  it("shows the shared error display", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("boom"));
    renderPage();
    expect(await screen.findByText("boom")).toBeInTheDocument();
  });
});
