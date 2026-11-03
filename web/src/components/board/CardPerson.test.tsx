import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardCardProvider } from "@/components/board/BoardCardProvider";
import { TicketCard } from "@/components/board/TicketCard";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import type { Ticket, TicketStatus } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  errorMessage: () => "",
}));

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
  const actual = await vi.importActual<typeof import("react-router")>("react-router");
  return { ...actual, useNavigate: () => mockNavigate };
});

const viewer = vi.hoisted(() => ({ canWrite: true }));
vi.mock("@/hooks/useActionState", () => ({
  useActionState: () => ({ state: viewer.canWrite ? "live" : "hidden" }),
}));

const people = [
  { user_id: "u-alice", login: "alice", display_name: "Alice", avatar_url: "" },
  { user_id: "u-lena", login: "lena", display_name: "Lena", avatar_url: "" },
];

const ticket = (overrides: Partial<Ticket> = {}): Ticket => ({
  id: "t-1",
  project_id: "p-1",
  category_id: "c-1",
  type_id: "",
  title: "Fix login",
  body: "",
  status: "open" as TicketStatus,
  position: 0,
  number: 142,
  doc_id: "",
  developer: "alice",
  developer_id: "u-alice",
  tester: "",
  tester_id: "",
  reporter: { kind: "user", login: "alice" },
  due_on: "",
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: [],
  ...overrides,
});

// The card renders from the board's cached list, as on the board, so an edit shows the moment the cache is patched.
const BoardCard = () => {
  const { data } = useFetchTicketsByProject("p-1");
  return data?.map((t) => <TicketCard key={t.id} ticket={t} />);
};

const renderBoard = (cached: Ticket) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  client.setQueryData(["getTicketsByProject", "p-1"], [cached]);
  render(
    <QueryClientProvider client={client}>
      <BoardCardProvider projectId="p-1">
        <BoardCard />
      </BoardCardProvider>
    </QueryClientProvider>,
  );
};

const statuses = [
  { id: "open", name: "Open", kind: "todo", icon: "Circle", position: 0 },
  { id: "qa", name: "QA", kind: "testing", icon: "Circle", position: 1 },
];

beforeEach(() => {
  viewer.canWrite = true;
  mockNavigate.mockReset();
  vi.mocked(api.patch).mockReset();
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url.endsWith("/people")) return { data: { people } };
    if (url === "/api/statuses") return { data: statuses };
    return { data: [] };
  });
});

describe("CardPerson", () => {
  it("reassigns the developer from the card's avatar without opening the ticket, showing the new face before the server answers", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockReturnValue(new Promise(() => {}));
    renderBoard(ticket());

    await user.click(await screen.findByRole("button", { name: "Developer: Alice" }));
    expect(screen.getByRole("button", { name: /Alice/, pressed: true })).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: /Lena/ }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/developer", { login: "lena" });
    expect(await screen.findByRole("button", { name: "Developer: Lena" })).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("reassigns the tester while the ticket sits in a testing column", async () => {
    const user = userEvent.setup();
    vi.mocked(api.patch).mockResolvedValue({ data: ticket({ status: "qa" as TicketStatus, tester: "lena", tester_id: "u-lena" }) });
    renderBoard(ticket({ status: "qa" as TicketStatus }));

    await user.click(await screen.findByRole("button", { name: "Tester: no one" }));
    await user.click(await screen.findByRole("button", { name: /Lena/ }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/tester", { login: "lena" });
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("shows a viewer who can't write the default face with no picker", async () => {
    viewer.canWrite = false;
    renderBoard(ticket({ developer: "", developer_id: "" }));

    expect(await screen.findByRole("img", { name: "Developer: no one" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Developer: no one" })).not.toBeInTheDocument();
  });
});
