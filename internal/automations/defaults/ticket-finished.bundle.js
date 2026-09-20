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

// automations/defaults/ticket-finished/index.ts
function ticketID(ticket) {
  const id = ticket?.id;
  return typeof id === "string" ? id : undefined;
}
var automation = defineAutomation({
  name: "Ticket finished",
  description: "Moves a ticket to its workspace's completed status once every linked PR has merged.",
  config: {
    completedStatusId: { type: "status", label: "Completed status", required: true }
  }
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
var ticket_finished_default = automation;
export {
  ticket_finished_default as default
};
