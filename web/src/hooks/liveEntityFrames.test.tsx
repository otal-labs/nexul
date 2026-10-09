import { act, render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import type { LiveSocket } from "@/api/ws";
import { useFetchDoc, useFetchDocs, useFetchDocsByProject, useFetchDocWatchers } from "@/hooks/DocHooks";
import { useFetchTicket, useFetchTicketLinks, useFetchTickets, useFetchTicketsByDoc, useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useFetchBlockers, useFetchTicketLinkSet } from "@/hooks/TicketLinkHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { useLiveEvents } from "@/hooks/useLiveEvents";
import type { Doc } from "@/models/Doc";
import type { Ticket } from "@/models/Ticket";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() }, errorMessage: vi.fn() }));

class FakeSocket implements LiveSocket {
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  close = vi.fn();
}

const ticket: Ticket = {
  id: "t-1",
  project_id: "p-1",
  category_id: "",
  type_id: "",
  title: "Login times out",
  body: "first draft",
  status: "open",
  position: 0,
  number: 1,
  doc_id: "doc-1",
  developer: "",
  tester: "",
  reporter: { kind: "user", login: "alice" },
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-01T10:00:00Z",
  labels: [],
};

const doc: Doc = {
  id: "d-1",
  project_id: "p-1",
  folder_id: "f-1",
  title: "Spec",
  body: "{}",
  version: 1,
  archived: false,
  locked: false,
  created_by: "alice",
  created_at: "2026-10-01T10:00:00Z",
  updated_at: "2026-10-01T10:00:00Z",
};

// The server as the browser reads it; a frame's change lands here first, as it does on the real server.
let server: { ticket: Ticket; doc: Doc };

const respond = (url: string) => {
  if (url === "/api/tickets") return [server.ticket];
  if (url === `/api/tickets/${ticket.id}`) return server.ticket;
  if (url === `/api/tickets/${ticket.id}/links`) return { prs: [], branches: [] };
  if (url === `/api/tickets/${ticket.id}/ticket-links`) return { found_in: null, origin_unknown: false, bugs_found: [], blocked_by: [], blocks: [], blocked: false };
  if (url === "/api/tickets/blockers") return {};
  if (url === "/api/plays/runs") return [];
  if (url === "/api/docs") return [{ ...server.doc, can_open: true }];
  if (url === `/api/docs/${doc.id}`) return server.doc;
  if (url === `/api/docs/${doc.id}/watchers`) return { watchers: [{ user_id: "alice", source: "auto", created_at: "" }], watching: true };
  throw new Error(`unexpected GET ${url}`);
};

// A board, a ticket page, and a doc's tickets section, all open at once.
const TicketViews = () => {
  useFetchTickets();
  useFetchTicket(ticket.id);
  useFetchTicketsByDoc(ticket.doc_id);
  useFetchTicketsByProject(ticket.project_id);
  useFetchTicketLinks(ticket.id);
  useFetchTicketLinkSet(ticket.id);
  useFetchBlockers();
  useFetchTrails("ticket", ticket.id);
  return null;
};

// The docs list, the project's doc tree, and an open doc page.
const DocViews = () => {
  useFetchDocs();
  useFetchDocsByProject(doc.project_id);
  useFetchDoc(doc.id);
  useFetchDocWatchers(doc.id);
  return null;
};

const Live = ({ wsFactory }: { wsFactory: () => LiveSocket }) => {
  useLiveEvents("ws://live/ws/events", { wsFactory });
  return null;
};

describe("a live ticket or doc frame", () => {
  let socket: FakeSocket;
  let client: QueryClient;

  beforeEach(() => {
    server = { ticket, doc };
    vi.mocked(api.get).mockReset();
    vi.mocked(api.get).mockImplementation(async (url: string) => ({ data: respond(url) }));
  });

  const mount = async (views: React.ReactNode) => {
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const sockets: FakeSocket[] = [];
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <Live
            wsFactory={() => {
              const s = new FakeSocket();
              sockets.push(s);
              return s;
            }}
          />
          {views}
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await vi.waitFor(() => expect(sockets[0]?.onmessage).toBeTruthy());
    socket = sockets[0]!;
    await vi.waitFor(() => expect(client.isFetching()).toBe(0));
    vi.mocked(api.get).mockClear();
  };

  // Requests the frame set off, counted once every refetch has settled.
  const requestsAfter = async (topic: string, payload: unknown) => {
    act(() => socket.onmessage?.({ data: JSON.stringify({ topic, type: "event", payload }) }));
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
    await vi.waitFor(() => expect(client.isFetching()).toBe(0));
    return vi.mocked(api.get).mock.calls.map(([url]) => url);
  };

  it("sends no request for a ticket's body commit, which a live editor saves every few seconds", async () => {
    await mount(<TicketViews />);
    server.ticket = { ...ticket, body: "second draft", updated_at: "2026-10-01T10:00:05Z" };
    expect(await requestsAfter("ticket.updated", { ticket: server.ticket })).toEqual([]);
  });

  it("shows a renamed ticket in all four ticket views: the board, its page, its doc's list, and its project's list", async () => {
    await mount(<TicketViews />);
    server.ticket = { ...ticket, title: "Login never times out", updated_at: "2026-10-01T10:00:05Z" };
    await requestsAfter("ticket.updated", { ticket: server.ticket });
    const titles = [
      client.getQueryData<Ticket[]>(["getTickets"])?.[0]?.title,
      client.getQueryData<Ticket>(["getTicket", ticket.id])?.title,
      client.getQueryData<Ticket[]>(["getTicketsByDoc", ticket.doc_id])?.[0]?.title,
      client.getQueryData<Ticket[]>(["getTicketsByProject", ticket.project_id])?.[0]?.title,
    ];
    expect(titles).toEqual(Array(4).fill("Login never times out"));
  });

  it("refetches only the board's blockers when a ticket changes column", async () => {
    await mount(<TicketViews />);
    server.ticket = { ...ticket, status: "in_progress", updated_at: "2026-10-01T10:00:05Z" };
    expect(await requestsAfter("ticket.status_changed", { ticket: server.ticket, from: "open", to: "in_progress" })).toEqual(["/api/tickets/blockers"]);
    expect(client.getQueryData<Ticket[]>(["getTicketsByProject", ticket.project_id])?.[0]?.status).toBe("in_progress");
  });

  it("sends no request for a doc's body commit by someone already watching it, and shows its new version", async () => {
    await mount(<DocViews />);
    server.doc = { ...doc, version: 2, updated_at: "2026-10-01T10:00:05Z" };
    expect(await requestsAfter("doc.updated", { doc: server.doc, actor_id: "alice" })).toEqual([]);
    expect(client.getQueryData<Doc>(["getDoc", doc.id])?.version).toBe(2);
    expect(client.getQueryData<Doc[]>(["getDocs", "byProject", doc.project_id])?.[0]?.version).toBe(2);
  });
});
