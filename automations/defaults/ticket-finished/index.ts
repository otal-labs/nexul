import { defineAutomation } from "@otal-labs/nexul-sdk/automation";
import { firstStatusOfKind } from "../board-statuses.ts";

// Narrows the fields this automation reads instead of trusting the whole payload.
function ticketRef(ticket: unknown): { id?: string; projectId?: string } {
  const { id, project_id } = (ticket ?? {}) as { id?: unknown; project_id?: unknown };
  return {
    ...(typeof id === "string" && id ? { id } : {}),
    ...(typeof project_id === "string" && project_id ? { projectId: project_id } : {}),
  };
}

const automation = defineAutomation({
  name: "Ticket finished",
  description: "Moves a ticket to its project's Done status once every linked PR has merged.",
});

automation.on("ticket.finished", async (payload, ctx) => {
  const { id, projectId } = ticketRef(payload.ticket);
  if (!id || !projectId) {
    ctx.log("skipped: missing ticket id or project id", { ticketId: id, projectId });
    return false;
  }
  const statusId = await firstStatusOfKind(ctx, projectId, "done");
  if (!statusId) {
    ctx.log("no done status in the ticket's project, nothing to move", { ticketId: id, projectId });
    return true;
  }
  await ctx.api.request("PATCH", `/api/tickets/${id}/status`, { status: statusId });
  ctx.log("moved ticket to its done status", { ticketId: id, statusId });
  return true;
});

export default automation;
