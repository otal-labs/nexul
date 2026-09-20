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

// automations/defaults/pr-opened/index.ts
function linkedTicketIDs(pr) {
  const ids = pr?.linked_ticket_ids;
  if (!Array.isArray(ids))
    return [];
  return ids.filter((id) => typeof id === "string");
}
var automation = defineAutomation({
  name: "PR opened",
  description: "Moves a ticket to its workspace's in-review status when a linked pull request opens.",
  config: {
    inReviewStatusId: { type: "status", label: "In-review status", required: true }
  }
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
var pr_opened_default = automation;
export {
  pr_opened_default as default
};
