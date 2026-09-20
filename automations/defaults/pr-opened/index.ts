import { defineAutomation } from "@nexul/sdk/automation";

// git.pr_opened's pr field is a loose object on the wire (internal/gitprovider.PR
// serialized generically per internal/integrations/catalog.go); linked_ticket_ids
// is the one field this automation reads off it (internal/gitprovider/model.go).
function linkedTicketIDs(pr: unknown): string[] {
  const ids = (pr as { linked_ticket_ids?: unknown } | null)?.linked_ticket_ids;
  if (!Array.isArray(ids)) return [];
  return ids.filter((id): id is string => typeof id === "string");
}

const automation = defineAutomation({
  name: "PR opened",
  description: "Moves a ticket to its workspace's in-review status when a linked pull request opens.",
  config: {
    inReviewStatusId: { type: "status", label: "In-review status", required: true },
  },
});

automation.on("git.pr_opened", async (payload, ctx) => {
  const ticketIds = linkedTicketIDs(payload.pr);
  if (ticketIds.length === 0) {
    ctx.log("no linked tickets on this PR, nothing to move");
    return true;
  }
  const statusId = ctx.config.inReviewStatusId;
  if (!statusId) {
    ctx.log("skipped: no in-review status configured", { ticketIds });
    return false;
  }
  for (const id of ticketIds) {
    await ctx.api.request("PATCH", `/api/tickets/${id}/status`, { status: statusId });
  }
  ctx.log("moved linked tickets to in-review status", { ticketIds, statusId });
  return true;
});

export default automation;
