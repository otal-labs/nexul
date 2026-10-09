import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { ticketChanged, ticketCreated, ticketRemoved } from "@/hooks/TicketCache";
import type { Ticket } from "@/models/Ticket";

const ticket = {
  id: "t-1",
  project_id: "p-1",
  doc_id: "doc-1",
  title: "Login times out",
  status: "open",
  updated_at: "2026-10-01T10:00:00Z",
} as Ticket;

const seeded = () => {
  const client = new QueryClient();
  client.setQueryData(["getTickets"], [ticket]);
  client.setQueryData(["getTicket", ticket.id], ticket);
  client.setQueryData(["getTicketsByDoc", "doc-1"], [ticket]);
  client.setQueryData(["getTicketsByDoc", "doc-2"], []);
  client.setQueryData(["getTicketsByProject", "p-1"], [ticket]);
  return client;
};

const stale = (client: QueryClient, key: unknown[]) => client.getQueryState(key)?.isInvalidated;

describe("the ticket cache", () => {
  it("moves a ticket whose source doc changed out of the old doc's list, and refetches the new doc's", async () => {
    const client = seeded();
    await ticketChanged(client, { ...ticket, doc_id: "doc-2", updated_at: "2026-10-01T10:00:05Z" });
    expect(client.getQueryData(["getTicketsByDoc", "doc-1"])).toEqual([]);
    expect(stale(client, ["getTicketsByDoc", "doc-2"])).toBe(true);
    expect(client.getQueryData<Ticket>(["getTicket", ticket.id])?.doc_id).toBe("doc-2");
  });

  it("keeps the newer copy when a frame from before the viewer's own save arrives after it", async () => {
    const client = seeded();
    await ticketChanged(client, { ...ticket, title: "Saved last", updated_at: "2026-10-01T10:00:09Z" });
    await ticketChanged(client, { ...ticket, title: "Saved first", updated_at: "2026-10-01T10:00:05Z" });
    expect(client.getQueryData<Ticket>(["getTicket", ticket.id])?.title).toBe("Saved last");
    expect(client.getQueryData<Ticket[]>(["getTickets"])?.[0]?.title).toBe("Saved last");
  });

  it("refetches the lists a new ticket belongs to and leaves the others alone", async () => {
    const client = seeded();
    await ticketCreated(client, { ...ticket, id: "t-2", doc_id: "doc-2" });
    expect(stale(client, ["getTickets"])).toBe(true);
    expect(stale(client, ["getTicketsByDoc", "doc-2"])).toBe(true);
    expect(stale(client, ["getTicketsByProject", "p-1"])).toBe(true);
    expect(stale(client, ["getTicketsByDoc", "doc-1"])).toBe(false);
  });

  it("drops a deleted ticket from every list and sends its open page to not found", async () => {
    const client = seeded();
    await ticketRemoved(client, ticket.id);
    for (const key of [["getTickets"], ["getTicketsByDoc", "doc-1"], ["getTicketsByProject", "p-1"]]) expect(client.getQueryData(key)).toEqual([]);
    expect(stale(client, ["getTicket", ticket.id])).toBe(true);
  });
});
