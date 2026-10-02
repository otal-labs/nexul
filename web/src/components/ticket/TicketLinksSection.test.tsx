import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import type { Ticket } from "@/models/Ticket";
import type { LinkedTicket, TicketLinkSet } from "@/models/TicketLink";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(() => "that would form a cycle"),
}));

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("sonner", () => ({ toast }));

const linked = (id: string, number: number, title: string, done = false): LinkedTicket => ({
  id,
  project_id: "p-1",
  prefix: "BKS",
  number,
  title,
  status: "open",
  done,
});

const emptySet: TicketLinkSet = {
  found_in: null,
  origin_unknown: false,
  bugs_found: [],
  blocked_by: [],
  blocks: [],
  blocked: false,
};

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

const statuses = [
  { id: "st-progress", name: "Doing", kind: "progress" },
  { id: "st-shipped", name: "Shipped", kind: "done" },
];

const mockApi = (set: TicketLinkSet) => {
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/tickets/t-1/ticket-links") return { data: set };
    if (url === "/api/statuses") return { data: statuses };
    if (url === "/api/ticket-types") {
      return { data: [{ id: "tt-task", name: "task", body_template: "" }, { id: "tt-bug", name: "Bug", body_template: "## Steps" }] };
    }
    if (url === "/api/tickets") {
      return { data: [{ id: "t-1", project_id: "p-1", number: 1, title: "frontend /books" }, { id: "t-2", project_id: "p-1", number: 2, title: "backend /books" }] };
    }
    if (url === "/api/projects") return { data: [{ id: "p-1", prefix: "BKS", name: "Books" }] };
    return { data: [] };
  });
};

const renderSection = (overrides: Partial<Ticket> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <TicketLinksSection ticket={{ ...ticket, ...overrides }} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.resetAllMocks();
});

describe("TicketLinksSection", () => {
  it("shows an empty row with no links", async () => {
    mockApi(emptySet);
    renderSection();
    expect(await screen.findByText("No blockers or found-in links.")).toBeInTheDocument();
  });

  it("shows both directions and marks which blockers are done", async () => {
    mockApi({
      ...emptySet,
      blocked_by: [linked("t-2", 2, "backend /books"), linked("t-3", 3, "auth", true)],
      blocks: [linked("t-4", 4, "mobile /books")],
      found_in: linked("t-5", 5, "books v1", true),
      blocked: true,
    });
    renderSection();
    expect(await screen.findByText("Blocked by")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /BKS-2/ })).toHaveAttribute("href", "/acme/tickets/BKS-2");
    expect(screen.getByRole("img", { name: "Not done yet" })).toBeInTheDocument();
    expect(screen.getAllByRole("img", { name: "Done" })).toHaveLength(2);
    expect(screen.getByText("Blocks")).toBeInTheDocument();
    expect(screen.getByText("Found in")).toBeInTheDocument();
  });

  it("shows only the key in a row and opens the title and description on keyboard focus", async () => {
    mockApi({ ...emptySet, blocks: [linked("t-4", 4, "mobile /books")] });
    const body = JSON.stringify({ type: "doc", content: [{ type: "paragraph", content: [{ type: "text", text: "Port the list to mobile." }] }] });
    const get = vi.mocked(api.get).getMockImplementation()!;
    vi.mocked(api.get).mockImplementation(async (url: string) =>
      url === "/api/tickets/t-4" ? { data: { ...ticket, id: "t-4", title: "mobile /books", body } } : get(url),
    );
    renderSection();
    const key = await screen.findByRole("link", { name: /BKS-4/ });
    expect(key).toHaveTextContent(/^BKS-4$/);
    act(() => key.focus());
    expect(await screen.findByText("Port the list to mobile.")).toBeInTheDocument();
    expect(screen.getByText("mobile /books")).toBeInTheDocument();
  });

  it("removes a blocker and a found-in link", async () => {
    mockApi({ ...emptySet, blocked_by: [linked("t-2", 2, "backend /books")], found_in: linked("t-5", 5, "books v1"), blocked: true });
    vi.mocked(api.delete).mockResolvedValue({ data: emptySet });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Remove link to BKS-2" }));
    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/blocked-by/t-2");
    await user.click(screen.getByRole("button", { name: "Remove link to BKS-5" }));
    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/found-in");
  });

  it("shows and removes the origin-unknown marker", async () => {
    mockApi({ ...emptySet, origin_unknown: true });
    vi.mocked(api.delete).mockResolvedValue({ data: emptySet });
    const user = userEvent.setup();
    renderSection();
    expect(await screen.findByText("Origin unknown")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Remove origin unknown" }));
    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/found-in");
  });

  it("adds a blocker picked from the ticket list, excluding the ticket itself", async () => {
    mockApi(emptySet);
    vi.mocked(api.post).mockResolvedValue({ data: emptySet });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Link a ticket" }));
    await user.click(screen.getByRole("button", { name: /Blocked by/ }));
    await user.type(screen.getByRole("textbox", { name: "Search tickets" }), "books");
    const options = await screen.findAllByRole("button", { name: /BKS-/ });
    expect(options).toHaveLength(1);
    expect(within(options[0]!).getByText("BKS-2")).toBeInTheDocument();
    await user.click(options[0]!);
    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/blocked-by", { blocker_id: "t-2" });
  });

  it("sets found-in from the picker and toasts a refused link", async () => {
    mockApi(emptySet);
    vi.mocked(api.put).mockRejectedValue(new Error("conflict"));
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Link a ticket" }));
    await user.click(screen.getByRole("button", { name: /Found in/ }));
    await user.click(await screen.findByRole("button", { name: /BKS-2/ }));
    expect(api.put).toHaveBeenCalledWith("/api/tickets/t-1/found-in", { origin_id: "t-2" });
    await vi.waitFor(() => expect(toast.error).toHaveBeenCalledWith("that would form a cycle"));
  });
});
