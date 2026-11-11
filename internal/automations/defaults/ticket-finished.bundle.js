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

// automations/defaults/ticket-finished/index.ts
function ticketRef(ticket) {
  const { id, project_id } = ticket ?? {};
  return {
    ...typeof id === "string" && id ? { id } : {},
    ...typeof project_id === "string" && project_id ? { projectId: project_id } : {}
  };
}
var automation = defineAutomation({
  name: "Ticket finished",
  description: "Moves a ticket to its project's Done status once every linked PR has merged."
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
var ticket_finished_default = automation;
export {
  ticket_finished_default as default
};
