import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketTestSection } from "@/components/ticket/TicketTestSection";
import type { Ticket } from "@/models/Ticket";
import type { TestTarget } from "@/models/TicketTest";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  errorMessage: vi.fn(() => "failed"),
}));

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast }));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "tt-feature",
  title: "Login page",
  body: "## Why\n\nNo login.\n\n## Acceptance criteria\n\nThe form signs you in\n\n## Out of scope\n\nSSO",
  status: "st-qa" as Ticket["status"],
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

const statuses = [
  { id: "st-build", name: "Build", kind: "progress" },
  { id: "st-qa", name: "QA", kind: "testing" },
  { id: "st-shipped", name: "Shipped", kind: "done" },
];

const mockApi = (target: TestTarget) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/statuses") return { data: statuses };
    if (url === "/api/tickets/t-1/test-target") return { data: target };
    if (url === "/api/stacks") return { data: [{ id: "s-1", name: "web" }] };
    return { data: [] };
  });
  vi.mocked(api.post).mockResolvedValue({ data: ticket });
};

const renderSection = (overrides: Partial<Ticket> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <TicketTestSection ticket={{ ...ticket, ...overrides }} />
        <ContextAwareConfirmation.ConfirmationRoot />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.resetAllMocks();
});

describe("TicketTestSection", () => {
  it("shows nothing outside a testing-stage column", async () => {
    mockApi({ url: "" });
    renderSection({ status: "st-build" as Ticket["status"] });
    await vi.waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/statuses", expect.anything()));
    expect(screen.queryByText("Test this")).not.toBeInTheDocument();
  });

  it("offers a shared environment with its label and the acceptance criteria", async () => {
    mockApi({ url: "https://qa.example.com", kind: "shared", branch: "dev" });
    renderSection();
    expect(await screen.findByRole("link", { name: "https://qa.example.com" })).toHaveAttribute("href", "https://qa.example.com");
    expect(screen.getByText("Shared, may include other changes.")).toBeInTheDocument();
    expect(await screen.findByText("The form signs you in")).toBeInTheDocument();
    expect(screen.queryByText("SSO")).not.toBeInTheDocument();
  });

  it("names the branch of a preview", async () => {
    mockApi({ url: "https://login.example.com", kind: "preview", branch: "feature/login" });
    renderSection();
    expect(await screen.findByText("feature/login")).toBeInTheDocument();
    expect(screen.queryByText("Shared, may include other changes.")).not.toBeInTheDocument();
  });

  it("says there is no test environment and points at the stack's deploy branches", async () => {
    mockApi({ url: "" });
    renderSection({ body: "" });
    expect(await screen.findByText(/No test environment yet/)).toBeInTheDocument();
    expect(await screen.findByRole("link", { name: "web" })).toHaveAttribute("href", "/stacks/s-1?section=branches");
    expect(screen.getByText(/no Acceptance criteria section/)).toBeInTheDocument();
  });

  it("passes the ticket", async () => {
    mockApi({ url: "" });
    renderSection();
    await userEvent.click(await screen.findByRole("button", { name: "Pass" }));
    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/test/pass");
    await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith("Passed and moved to done"));
  });

  it("fails the ticket with the bug template's sections", async () => {
    mockApi({ url: "" });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Fail" }));
    await user.click(await screen.findByRole("button", { name: "Send back" }));
    expect(await screen.findByText("Say what went wrong")).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText("Steps to reproduce"), "Open /login");
    await user.type(screen.getByLabelText("Expected result"), "A form");
    await user.type(screen.getByLabelText("Actual result"), "Blank page");
    await user.click(screen.getByRole("button", { name: "Send back" }));

    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/test/fail", {
        steps: "Open /login",
        expected: "A form",
        actual: "Blank page",
        screenshots: [],
      }),
    );
  });

  it("attaches a screenshot to the ticket and sends its id", async () => {
    mockApi({ url: "" });
    vi.mocked(api.post).mockImplementation(async (url: string) => {
      if (url === "/api/attachments") return { data: { id: "att-1", name: "shot.png" } };
      return { data: ticket };
    });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Fail" }));
    await user.upload(await screen.findByLabelText("Add screenshot"), new File(["x"], "shot.png", { type: "image/png" }));
    expect(await screen.findByText("shot.png")).toBeInTheDocument();
    const form = vi.mocked(api.post).mock.calls[0]?.[1] as FormData;
    expect(form.get("ticket_id")).toBe("t-1");

    await user.type(screen.getByLabelText("Actual result"), "Blank page");
    await user.click(screen.getByRole("button", { name: "Send back" }));
    await vi.waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/test/fail", expect.objectContaining({ screenshots: ["att-1"] })),
    );
  });
});
