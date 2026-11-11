// sdk/src/define-automation.ts
class Automation {
  name;
  description;
  configSchema;
  handlers = new Map;
  constructor(opts) {
    if (!opts.name.trim())
      throw new Error("defineAutomation requires a non-empty name");
    this.name = opts.name;
    this.description = opts.description;
    this.configSchema = opts.config ?? {};
  }
  on(topic, handler) {
    this.handlers.set(topic, handler);
    return this;
  }
  getHandler(topic) {
    return this.handlers.get(topic);
  }
  get subscriptions() {
    return [...this.handlers.keys()];
  }
}
function defineAutomation(opts) {
  return new Automation(opts);
}

// automations/defaults/board-statuses.ts
function statusFields(raw) {
  if (!Array.isArray(raw))
    return [];
  return raw.flatMap((s) => {
    const { id, kind, position } = s ?? {};
    return typeof id === "string" && typeof kind === "string" && typeof position === "number" ? [{ id, kind, position }] : [];
  });
}
async function firstStatusOfKind(ctx, projectId, kind) {
  const raw = await ctx.api.request("GET", `/api/statuses?project_id=${encodeURIComponent(projectId)}`);
  const matches = statusFields(raw).filter((s) => s.kind === kind);
  return matches.sort((a, b) => a.position - b.position)[0]?.id;
}

// automations/defaults/pr-opened/index.ts
function linkedTicketIDs(pr) {
  const ids = pr?.linked_ticket_ids;
  if (!Array.isArray(ids))
    return [];
  return ids.filter((id) => typeof id === "string");
}
function projectID(ticket) {
  const id = ticket?.project_id;
  return typeof id === "string" && id ? id : undefined;
}
var automation = defineAutomation({
  name: "PR opened",
  description: "Moves each linked ticket to its project's In review status when a pull request opens."
});
automation.on("git.pr_opened", async (payload, ctx) => {
  const ticketIds = linkedTicketIDs(payload.pr);
  if (ticketIds.length === 0) {
    ctx.log("no linked tickets on this PR, nothing to move");
    return true;
  }
  let handled = true;
  const reviewStatusByProject = new Map;
  for (const id of ticketIds) {
    const projectId = projectID(await ctx.api.request("GET", `/api/tickets/${id}`));
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
var pr_opened_default = automation;
export {
  pr_opened_default as default
};
