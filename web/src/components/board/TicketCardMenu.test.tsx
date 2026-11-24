import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { dayKey } from "@nexul/client-core/calendar";

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

const confirmDialog = vi.hoisted(() => ({ answer: true, open: vi.fn() }));
vi.mock("@/hooks/useConfirmationDialog", () => ({
  useConfirmationDialog: () => ({ open: confirmDialog.open }),
}));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "",
  title: "Fix login",
  body: "",
  status: "open" as TicketStatus,
  position: 0,
  number: 142,
  doc_id: "",
  developer: "",
  developer_id: "",
  tester: "",
  tester_id: "",
  reporter: { kind: "user", login: "alice" },
  due_on: "",
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
  labels: ["Backend"],
};

const statuses = [
  { id: "open", name: "Open", kind: "backlog", icon: "Circle", position: 0 },
  { id: "doing", name: "Doing", kind: "progress", icon: "Circle", position: 1 },
];

const BoardCard = () => {
  const { data } = useFetchTicketsByProject("p-1");
  return data?.map((t) => <TicketCard key={t.id} ticket={t} />);
};

const renderBoard = (overrides: Partial<Ticket> = {}) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  client.setQueryData(["getTicketsByProject", "p-1"], [{ ...ticket, ...overrides }]);
  client.setQueryData(["getProject", "p-1"], { id: "p-1", prefix: "WEB" });
  render(
    <QueryClientProvider client={client}>
      <BoardCardProvider projectId="p-1">
        <BoardCard />
      </BoardCardProvider>
    </QueryClientProvider>,
  );
};

const openMenu = async () => {
  fireEvent.contextMenu(screen.getByText("Fix login"));
  return screen.findByRole("menu");
};

// Opens the submenu the way a keyboard does, which is also the path a screen reader takes.
const openSub = async (name: RegExp) => {
  const trigger = await screen.findByRole("menuitem", { name });
  trigger.focus();
  fireEvent.keyDown(trigger, { key: "ArrowRight" });
  return (await screen.findAllByRole("menu")).at(-1)!;
};

beforeEach(() => {
  viewer.canWrite = true;
  confirmDialog.answer = true;
  confirmDialog.open.mockReset().mockImplementation(async () => confirmDialog.answer);
  mockNavigate.mockReset();
  for (const method of ["patch", "post", "delete"] as const) vi.mocked(api[method]).mockReset().mockResolvedValue({ data: ticket });
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedWorkspaceSlug: "acme" });
  vi.mocked(api.get).mockImplementation(async (url: string) => {
    if (url === "/api/statuses") return { data: statuses };
    if (url === "/api/tickets/labels") return { data: ["Backend", "Frontend"] };
    if (url.endsWith("/people")) return { data: { people: [] } };
    return { data: [] };
  });
});

describe("TicketCardMenu", () => {
  it("moves the ticket to the status picked from the card's right-click menu, without opening the ticket", async () => {
    renderBoard();
    await openMenu();
    const sub = await openSub(/^Status/);

    await userEvent.click(await within(sub).findByRole("menuitemradio", { name: "Doing" }));

    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/status", { status: "doing" }, expect.anything());
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it("toggles labels in place, keeping the menu open for the next one", async () => {
    renderBoard();
    await openMenu();
    const sub = await openSub(/^Labels/);

    await userEvent.click(await within(sub).findByRole("menuitemcheckbox", { name: "Frontend" }));
    await userEvent.click(within(sub).getByRole("menuitemcheckbox", { name: "Backend" }));

    expect(api.post).toHaveBeenCalledWith("/api/tickets/t-1/labels", { label: "Frontend" });
    expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1/labels/Backend");
    expect(screen.getAllByRole("menu").length).toBeGreaterThan(0);
  });

  it("saves a quick due date once, as the local day it names, and closes the menu", async () => {
    renderBoard({ due_on: "2026-01-05" });
    await openMenu();
    const sub = await openSub(/^Due date/);
    const now = new Date();

    await userEvent.click(within(sub).getByRole("menuitem", { name: /^Tomorrow/ }));

    await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
    expect(api.patch).toHaveBeenCalledTimes(1);
    expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/due-date", {
      due_on: dayKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)),
    });
  });

  it("saves a typed due date on Enter, not on each change while the year is half typed", async () => {
    renderBoard();
    await openMenu();
    const sub = await openSub(/^Due date/);
    const field = within(sub).getByLabelText("Due date");

    fireEvent.change(field, { target: { value: "0202-12-03" } });
    fireEvent.change(field, { target: { value: "2026-12-03" } });
    expect(api.patch).not.toHaveBeenCalled();
    fireEvent.keyDown(field, { key: "Enter" });

    await waitFor(() => expect(api.patch).toHaveBeenCalledWith("/api/tickets/t-1/due-date", { due_on: "2026-12-03" }));
    expect(api.patch).toHaveBeenCalledTimes(1);
  });

  it("deletes the ticket only once the confirmation is accepted", async () => {
    renderBoard();
    confirmDialog.answer = false;
    await openMenu();
    await userEvent.click(screen.getByRole("menuitem", { name: "Delete" }));
    await waitFor(() => expect(confirmDialog.open).toHaveBeenCalledWith(expect.objectContaining({ title: "Delete WEB-142?" })));
    expect(api.delete).not.toHaveBeenCalled();

    confirmDialog.answer = true;
    await openMenu();
    await userEvent.click(screen.getByRole("menuitem", { name: "Delete" }));
    await waitFor(() => expect(api.delete).toHaveBeenCalledWith("/api/tickets/t-1"));
  });

  it("copies the ticket's full link", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    renderBoard();
    await openMenu();

    await userEvent.click(screen.getByRole("menuitem", { name: "Copy link" }));

    expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/acme/tickets/WEB-142`);
  });

  it("offers a viewer who can't write the ticket only Open and Copy link", async () => {
    viewer.canWrite = false;
    renderBoard();
    const menu = await openMenu();

    expect(within(menu).getAllByRole("menuitem").map((item) => item.textContent)).toEqual(["Open ticket", "Copy link"]);
  });
});
