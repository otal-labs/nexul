import { defineAutomation } from "@otal-labs/nexul-sdk/automation";
import { firstStatusOfKind } from "../board-statuses.ts";

// Narrows linked_ticket_ids, the one field this automation reads, which may be null on a PR that names no ticket.
function linkedTicketIDs(pr: unknown): string[] {
  const ids = (pr as { linked_ticket_ids?: unknown } | null)?.linked_ticket_ids;
  if (!Array.isArray(ids)) return [];
  return ids.filter((id): id is string => typeof id === "string");
}

function projectID(ticket: unknown): string | undefined {
  const id = (ticket as { project_id?: unknown } | null)?.project_id;
  return typeof id === "string" && id ? id : undefined;
}

const automation = defineAutomation({
  name: "PR opened",
  description: "Moves each linked ticket to its project's In review status when a pull request opens.",
});

automation.on("git.pr_opened", async (payload, ctx) => {
  const ticketIds = linkedTicketIDs(payload.pr);
  if (ticketIds.length === 0) {
    ctx.log("no linked tickets on this PR, nothing to move");
    return true;
  }
  let handled = true;
  const reviewStatusByProject = new Map<string, string | undefined>();
  for (const id of ticketIds) {
    const projectId = projectID(await ctx.api.request<unknown>("GET", `/api/tickets/${id}`));
    if (!projectId) {
      ctx.log("skipped: ticket has no project id", { ticketId: id });
      handled = false;
      continue;
    }
    if (!reviewStatusByProject.has(projectId)) {
      reviewStatusByProject.set(projectId, await firstStatusOfKind(ctx, projectId, "review"));
    }
    const statusId = reviewStatusByProject.get(projectId);
    if (!statusId) {
      ctx.log("no review status in the ticket's project, nothing to move", { ticketId: id, projectId });
      continue;
    }
    await ctx.api.request("PATCH", `/api/tickets/${id}/status`, { status: statusId });
    ctx.log("moved ticket to its review status", { ticketId: id, statusId });
  }
  return handled;
});

export default automation;
