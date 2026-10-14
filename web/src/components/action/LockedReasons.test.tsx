import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { BoardCreateMenu } from "@/components/board/BoardCreateMenu";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketStatus, type Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asReader, settle } from "@/test/viewer";

vi.mock("@/api/client", () => ({ api: { get: vi.fn(), post: vi.fn(), patch: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "ticket-type-task",
  title: "Write migrations",
  body: "",
  status: TicketStatus.InProgress,
  position: 0,
  number: 1,
  doc_id: "",
  developer: "alice",
  tester: "",
  reporter: { kind: "user", login: "alice" },
  labels: ["backend"],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const ticketTypes = [{ id: "ticket-type-task", name: "task", position: 0, color: "", created_at: "", updated_at: "" }];

interface Surface {
  name: string;
  ui: ReactElement;
  // Opens a menu the controls sit in.
  open?: string;
  controls: { name: string | RegExp; reason: string }[];
}

// A locked reason never reads as the catalog label ("create and update ..."), which only the roles editor wants.
const surfaces: Surface[] = [
  {
    name: "the board's Create menu",
    ui: <BoardCreateMenu onNewTicket={vi.fn()} onNewCategory={vi.fn()} />,
    open: "Create",
    controls: [
      { name: "New ticket", reason: "Your role can't create tickets" },
      { name: "Report a bug", reason: "Your role can't report bugs" },
      { name: "New category", reason: "Your role can't add a category" },
    ],
  },
  {
    name: "a ticket's properties",
    ui: <TicketPropertiesPanel ticket={ticket} onTransition={vi.fn()} onSetType={vi.fn()} onAddLabel={vi.fn()} onRemoveLabel={vi.fn()} />,
    controls: [
      { name: /^Status:/, reason: "Your role can't change a ticket's status" },
      { name: /^Category:/, reason: "Your role can't change a ticket's category" },
      { name: /^Developer:/, reason: "Your role can't assign tickets" },
      { name: /^Type:/, reason: "Your role can't change a ticket's type" },
    ],
  },
];

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1", selectedProjectId: "p-1" });
  vi.mocked(api.get).mockImplementation(
    asReader({ showLocked: true, permissions: ["tickets:read", "projects:read"] }, (url: string) => (url === "/api/ticket-types" ? ticketTypes : [])),
  );
});

describe("a locked control's reason", () => {
  it.each(surfaces)("names what the viewer can't do on $name", async ({ ui, open, controls }) => {
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>{ui}</MemoryRouter>
      </QueryClientProvider>,
    );
    await settle();
    if (open) await userEvent.setup().click(await screen.findByRole("button", { name: open }));

    for (const { name, reason } of controls) {
      const control = await screen.findByRole("button", { name });
      expect(control).toHaveAttribute("aria-disabled", "true");
      expect(control).toHaveAccessibleDescription(reason);
    }
  });
});
