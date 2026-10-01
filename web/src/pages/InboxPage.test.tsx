import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { InboxPage } from "@/pages/InboxPage";
import { useInboxFolderStore } from "@/stores/inboxFolderStore";
import { useInboxStore } from "@/stores/inboxStore";

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
  workspace_id: "ws-1",
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
  workspace_id: "ws-1",
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

const mockApiResponse = (url: string) => {
    if (url === "/api/notifications") return Promise.resolve({ data: [ticketNotification, docNotification] });
    if (url === "/api/tickets/t-1") return Promise.resolve({ data: ticketData });
    if (url === "/api/tickets/t-1/ticket-links") return Promise.resolve({ data: { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false } });
    if (url === "/api/docs/doc-1") return Promise.resolve({ data: docData });
    if (url === "/api/pairing/presence") return Promise.resolve({ data: { computers: {} } });
    return Promise.resolve({ data: [] });
};

const mockApi = () => vi.mocked(api.get).mockImplementation(mockApiResponse);

beforeEach(() => {
  vi.mocked(api.get).mockReset();
  vi.mocked(api.post).mockReset();
  useInboxStore.setState({ selectedKey: null });
  useInboxFolderStore.setState({ collapsed: [] });
});

describe("InboxPage", () => {
  it("shows the first notification's source selected by default", async () => {
    mockApi();
    renderPage();
    expect(await screen.findByRole("heading", { name: "Write migrations" })).toBeInTheDocument();
    const list = screen.getByRole("navigation", { name: "Notifications" });
    expect(within(list).getByRole("button", { name: /Write migrations/ })).toHaveAttribute("aria-current", "true");
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

  it("opens a doc grouped under its folder and marks every unread notification about it read", async () => {
    const inGetSource = { subject_type: "doc", subject_id: "doc-ep07", subject_title: "GetSource EP07: Bluesky feeds", folder_id: "f-gs", folder_name: "GetSource", folder_is_default: false };
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/notifications")
        return Promise.resolve({
          data: [
            ticketNotification,
            { ...docNotification, ...inGetSource, id: "u2", kind: "doc.updated", read: false, created_at: "2026-08-12T10:00:00Z" },
            { ...docNotification, ...inGetSource, id: "u1", kind: "doc.updated", read: true, created_at: "2026-08-12T09:00:00Z" },
            { ...docNotification, ...inGetSource, id: "c1", kind: "doc.created", read: false, created_at: "2026-08-12T08:00:00Z" },
          ],
        });
      if (url === "/api/docs/doc-ep07") return Promise.resolve({ data: { ...docData, id: "doc-ep07", title: "GetSource EP07: Bluesky feeds" } });
      return mockApiResponse(url);
    });
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderPage();

    const folder = await screen.findByRole("region", { name: "GetSource" });
    expect(within(folder).getByRole("button", { name: /GetSource/ })).toHaveTextContent("1 doc · 3 updates");
    await user.click(within(folder).getByRole("button", { name: /EP07: Bluesky feeds/ }));

    expect(await screen.findByRole("heading", { name: "GetSource EP07: Bluesky feeds" })).toBeInTheDocument();
    expect(vi.mocked(api.post).mock.calls.map(([url]) => url).sort()).toEqual([
      "/api/notifications/c1/read",
      "/api/notifications/u2/read",
    ]);
  });

  it("hides a folder's docs when it is collapsed", async () => {
    vi.mocked(api.get).mockImplementation((url: string) => {
      if (url === "/api/notifications")
        return Promise.resolve({ data: [ticketNotification, { ...docNotification, subject_title: "GetSource EP01", folder_id: "f-gs", folder_name: "GetSource", folder_is_default: false }] });
      return mockApiResponse(url);
    });
    const user = userEvent.setup();
    renderPage();

    const folder = await screen.findByRole("region", { name: "GetSource" });
    expect(within(folder).getByText("EP01")).toBeInTheDocument();
    await user.click(within(folder).getByRole("button", { expanded: true }));
    expect(within(folder).queryByText("EP01")).not.toBeInTheDocument();
  });

  it("marks all notifications read", async () => {
    mockApi();
    vi.mocked(api.post).mockResolvedValue({ data: undefined });
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Mark all read" }));
    expect(api.post).toHaveBeenCalledWith("/api/notifications/read-all", undefined, expect.anything());
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
