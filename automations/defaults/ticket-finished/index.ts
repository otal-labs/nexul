import { defineAutomation } from "@nexul/sdk/automation";

// ticket.finished's payload types its ticket as a loose object (the event
// schema declares it "type: object" with no nested shape, see
// internal/integrations/catalog.go) — narrow just the field this automation
// needs instead of trusting the whole shape.
function ticketID(ticket: unknown): string | undefined {
  const id = (ticket as { id?: unknown } | null)?.id;
  return typeof id === "string" ? id : undefined;
}

const automation = defineAutomation({
  name: "Ticket finished",
  description: "Moves a ticket to its workspace's completed status once every linked PR has merged.",
  config: {
    completedStatusId: { type: "status", label: "Completed status", required: true },
  },
});

automation.on("ticket.finished", async (payload, ctx) => {
  const id = ticketID(payload.ticket);
  const statusId = ctx.config.completedStatusId;
  if (!id || !statusId) {
    ctx.log("skipped: missing ticket id or no completed status configured", { ticketId: id, statusId });
    return false;
  }
  await ctx.api.request("PATCH", `/api/tickets/${id}/status`, { status: statusId });
  ctx.log("moved ticket to completed status", { ticketId: id, statusId });
  return true;
});

export default automation;
