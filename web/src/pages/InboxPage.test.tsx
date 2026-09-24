import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { InboxPage } from "@/pages/InboxPage";

vi.mock("@/components/doc/collab/useCollabSession", () => ({
  useCollabSession: () => null,
}));

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn((error: unknown) => (error as Error)?.message ?? "Something went wrong"),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticketNotification = {
  id: "n1",
  user_id: "u1",
  kind: "ticket.assigned",
  subject_type: "ticket",
  subject_id: "t-1",
  subject_title: "Write migrations",
  read: false,
  created_at: "2026-08-12T12:00:00Z",
};

const docNotification = {
  id: "n2",
  user_id: "u1",
  kind: "doc.updated",
  subject_type: "doc",
  subject_id: "doc-1",
  subject_title: "Spec",
  read: true,
  created_at: "2026-08-12T11:00:00Z",
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
  doc_id: "",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "onik97" },
  labels: [],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const docData = {
  id: "doc-1",
  project_id: "p-1",
  title: "Spec",
  body: "The plan.",
  version: 1,
  archived: false,
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InboxPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const mockApi = () =>
  vi.mocked(api.get).mockImplementation((url: string) => {
    if (url === "/api/notifications") return Promise.resolve({ data: [ticketNotification, docNotification] });
    if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
    if (url === "/api/docs/doc-1") return Promise.resolve({ data: docData });
    if (url === "/api/pairing/presence") return Promise.resolve({ data: { computers: {} } });
    return Promise.resolve({ data: [] });
  });

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
});

describe("InboxPage", () => {
  it("shows the first notification's source selected by default", async () => {
    mockApi();
    renderPage();
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
  });

  it("marks an unread notification read and loads its source when selected", async () => {
    mockApi();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderPage();

    await screen.findByText("Spec");
    await user.click(within(screen.getByRole("navigation", { name: "Notifications" })).getByText("Spec"));

    expect(await screen.findByRole("heading", { name: "Spec" })).toBeInTheDocument();
    expect(api.post).not.toHaveBeenCalled();
  });

  it("marks read when selecting an unread notification", async () => {
    mockApi();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderPage();

    await screen.findByRole("heading", { name: "Write migrations" });
    const list = screen.getByRole("navigation", { name: "Notifications" });
    await user.click(within(list).getByRole("button", { name: /Write migrations/ }));
    expect(api.post).toHaveBeenCalledWith("/api/notifications/n1/read");
  });

  it("marks all notifications read", async () => {
    mockApi();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Mark all read" }));
    expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all");
  });

  it("shows the empty state when there are no notifications", async () => {
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/notifications") return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
    renderPage();
    expect(await screen.findByText("No notifications")).toBeInTheDocument();
  });
});
