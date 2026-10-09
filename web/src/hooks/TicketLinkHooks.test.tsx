import { describe, expect, it } from "vitest";

import { stageMoved, ticketLinkFollower } from "@/hooks/TicketLinkHooks";
import type { TicketStatus } from "@/models/Ticket";
import type { LinkedTicket, TicketLinkSet } from "@/models/TicketLink";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const linked = (id: string, status: string): LinkedTicket => ({ id, project_id: "p-1", prefix: "ACME", number: 1, title: id, status: status as TicketStatus, done: false });

const set = (blockedBy: LinkedTicket[] = []): TicketLinkSet => ({
  found_in: null,
  origin_unknown: false,
  bugs_found: [],
  blocked_by: blockedBy,
  blocks: [],
  blocked: blockedBy.length > 0,
});

describe("the ticket link follower", () => {
  it("refetches both ends of a new link, and the board's blockers only when one end now waits on the other", async () => {
    const client = seeded([
      [["getTicketLinkSet", "t-1"], set()],
      [["getTicketLinkSet", "t-2"], set()],
      [["getTicketLinkSet", "t-3"], set()],
      [["getBlockers"], {}],
    ]);
    await followFrame(ticketLinkFollower, "ticket.link_created", { link: { ticket_id: "t-1", kind: "found_in", target_id: "t-2" } }, client);
    expect([["t-1"], ["t-2"], ["t-3"]].map(([id]) => isStale(client, ["getTicketLinkSet", id]))).toEqual([true, true, false]);
    expect(isStale(client, ["getBlockers"])).toBe(false);

    await followFrame(ticketLinkFollower, "ticket.link_deleted", { link: { ticket_id: "t-1", kind: "blocked_by", target_id: "t-2" } }, client);
    expect(isStale(client, ["getBlockers"])).toBe(true);
  });

  it("refetches only the link sets naming a ticket in a column whose stage moved", async () => {
    const client = seeded([
      [["getTicketLinkSet", "t-1"], set([linked("t-2", "st-review")])],
      [["getTicketLinkSet", "t-3"], set([linked("t-4", "st-backlog")])],
      [["getBlockers"], {}],
    ]);
    await stageMoved(client, "st-review");
    expect([isStale(client, ["getTicketLinkSet", "t-1"]), isStale(client, ["getTicketLinkSet", "t-3"]), isStale(client, ["getBlockers"])]).toEqual([
      true,
      false,
      true,
    ]);
  });
});
