import { describe, expect, it } from "vitest";

import { ticketTypeFollower } from "@/hooks/TicketTypeHooks";
import type { TicketType } from "@/models/TicketType";
import { followFrame, isStale, seeded } from "@/test/followFrame";

const bug: TicketType = { id: "tt-1", name: "bug", position: 0, color: "", body_template: "", created_at: "", updated_at: "" };

describe("the ticket type follower", () => {
  it("patches a renamed type into its project's list, and refetches only that list for a new one", async () => {
    const client = seeded([
      [["getProjectTicketTypes", "p-1"], [bug]],
      [["getProjectTicketTypes", "p-2"], [bug]],
    ]);
    await followFrame(ticketTypeFollower, "ticket_type.updated", { ticket_type: { ...bug, project_id: "p-1", name: "defect" } }, client);
    expect(client.getQueryData<TicketType[]>(["getProjectTicketTypes", "p-1"])?.[0]?.name).toBe("defect");
    expect(isStale(client, ["getProjectTicketTypes", "p-1"])).toBe(false);

    await followFrame(ticketTypeFollower, "ticket_type.created", { ticket_type: { ...bug, id: "tt-2", project_id: "p-1" } }, client);
    expect([isStale(client, ["getProjectTicketTypes", "p-1"]), isStale(client, ["getProjectTicketTypes", "p-2"])]).toEqual([true, false]);
  });
});
