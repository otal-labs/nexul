import { describe, expect, test } from "bun:test";
import { createMockContext, eventFixtures } from "@nexul/sdk/testing";
import automation from "./index.ts";

describe("ticket-finished default", () => {
  test("moves the ticket to the configured completed status", async () => {
    const ctx = createMockContext(automation.configSchema, { config: { completedStatusId: "status-done" } });
    const payload = { ticket: { ...eventFixtures["ticket.finished"].ticket, id: "ticket-1" } };

    const result = await automation.getHandler("ticket.finished")!(payload, ctx);

    expect(result).toBe(true);
    expect(ctx.calls).toEqual([{ method: "PATCH", path: "/api/tickets/ticket-1/status", body: { status: "status-done" } }]);
  });

  test("fails without a configured completed status, and calls nothing", async () => {
    const ctx = createMockContext(automation.configSchema);
    const payload = { ticket: { ...eventFixtures["ticket.finished"].ticket, id: "ticket-1" } };

    const result = await automation.getHandler("ticket.finished")!(payload, ctx);

    expect(result).toBe(false);
    expect(ctx.calls).toEqual([]);
  });

  test("fails when the event carries no ticket id", async () => {
    const ctx = createMockContext(automation.configSchema, { config: { completedStatusId: "status-done" } });

    const result = await automation.getHandler("ticket.finished")!(eventFixtures["ticket.finished"], ctx);

    expect(result).toBe(false);
    expect(ctx.calls).toEqual([]);
  });
});
