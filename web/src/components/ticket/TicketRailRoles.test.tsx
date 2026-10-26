import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DevelopmentSection } from "@/components/ticket/DevelopmentSection";
import { TicketBugsSection } from "@/components/ticket/TicketBugsSection";
import { TicketLinksSection } from "@/components/ticket/TicketLinksSection";
import { TicketPropertiesPanel } from "@/components/ticket/TicketPropertiesPanel";
import { TicketTestSection } from "@/components/ticket/TicketTestSection";
import type { Ticket } from "@/models/Ticket";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { asViewer, memberWho } from "@/test/viewer";

vi.mock("@/api/client", () => ({
  api: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn(), delete: vi.fn() },
  errorMessage: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "tt-task",
  title: "Write migrations",
  body: "",
  status: "st-testing",
  position: 0,
  number: 1,
  doc_id: "doc-1",
  developer: "alice",
  tester: "",
  developer_id: "",
  tester_id: "",
  reporter: { kind: "user", login: "bob" },
  due_on: "",
  labels: ["backend"],
  created_at: "2026-08-02T12:00:00Z",
  updated_at: "2026-08-02T12:00:00Z",
};

const data = (url: string): unknown => {
  if (url === "/api/tickets/t-1") return ticket;
  if (url === "/api/statuses") return [{ id: "st-testing", name: "Testing", kind: "testing", icon: "", position: 0 }];
  if (url === "/api/ticket-types") return [{ id: "tt-task", name: "task", position: 0, color: "", body_template: "" }];
  if (url === "/api/tickets/t-1/links") return { prs: [], branches: [{ owner: "acme", repo: "app", branch: "ticket/1" }] };
  if (url === "/api/tickets/t-1/ticket-links") {
    const other = { id: "t-2", project_id: "p-1", prefix: "BE", number: 2, title: "Other", status: "open", done: false };
    return { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [other], blocks: [], blocked: true };
  }
  if (url === "/api/attachments") return [{ id: "a-1", ticket_id: "t-1", name: "shot.png", content_type: "text/plain", size: 10, uploaded_by: "bob", created_at: "" }];
  if (url === "/api/docs/doc-1") return { id: "doc-1", project_id: "p-1", title: "Spec" };
  if (url === "/api/tickets/labels") return ["backend"];
  return [];
};

const renderRail = () =>
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <TicketPropertiesPanel ticket={ticket} onTransition={vi.fn()} onSetType={vi.fn()} onAddLabel={vi.fn()} onRemoveLabel={vi.fn()}>
          <DevelopmentSection ticketId="t-1" />
          <AttachmentsSection owner={{ ticket_id: "t-1" }} />
          <TicketLinksSection ticket={ticket} />
          <TicketTestSection ticket={ticket} />
          <TicketBugsSection ticket={ticket} />
        </TicketPropertiesPanel>
      </MemoryRouter>
    </QueryClientProvider>,
  );

// Every control of the ticket page's rail that writes the ticket, and the value or text a reader still sees where it was.
const controls = [
  { name: "Status: Testing", stays: "Testing" },
  { name: "Category: No category", stays: "No category" },
  { name: "Developer: alice", stays: "alice" },
  { name: "Tester: no one", stays: "No one" },
  { name: "Due: none", stays: "No due date" },
  { name: "Type: task", stays: "task" },
  { name: "Add label", stays: "backend" },
  { name: "Remove label backend", stays: "backend" },
  { name: "Link development item", stays: "ticket/1" },
  { name: "Unlink branch ticket/1", stays: "ticket/1" },
  { name: "Add attachment", stays: "shot.png" },
  { name: "Delete shot.png", stays: "shot.png" },
  { name: "Add a link", stays: "Spec" },
  { name: "Remove source doc", stays: "Spec" },
  { name: "Remove link to BE-2", stays: "BE-2" },
  { name: "Pass", stays: "Acceptance criteria" },
  { name: "Fail", stays: "Acceptance criteria" },
  { name: "Report a bug", stays: "No bugs reported." },
];

describe("a ticket's rail for a member who may read it and not write it", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
    vi.mocked(api.get).mockReset();
  });

  it("keeps what each control showed and offers none of them", async () => {
    vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:read", "docs:read"], false), data));
    renderRail();

    expect(await screen.findByText("Spec")).toBeInTheDocument();
    expect(await screen.findByText("BE-2")).toBeInTheDocument();
    for (const { name, stays } of controls) {
      expect(screen.queryByRole("button", { name }), name).not.toBeInTheDocument();
      expect(screen.queryAllByText(stays, { exact: false }).length, stays).toBeGreaterThan(0);
    }
  });

  it("locks every one of them when a role of theirs shows locked actions", async () => {
    vi.mocked(api.get).mockImplementation(asViewer(memberWho(["tickets:read", "docs:read"], true), data));
    renderRail();

    for (const { name } of controls) expect(await screen.findByRole("button", { name }), name).toHaveAttribute("aria-disabled", "true");
  });
});
