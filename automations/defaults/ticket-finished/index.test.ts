import { describe, expect, test } from "bun:test";
import { ApiError } from "@otal-labs/nexul-sdk/api-client";
import { createMockContext, eventFixtures } from "@otal-labs/nexul-sdk/testing";
import automation from "./index.ts";

const handler = automation.getHandler("ticket.finished")!;
const finished = (ticket: Record<string, unknown>) => ({ ticket: { ...eventFixtures["ticket.finished"].ticket, ...ticket } });
const statuses = [
  { id: "status-later-done", kind: "done", position: 5 },
  { id: "status-review", kind: "review", position: 3 },
  { id: "status-done", kind: "done", position: 4 },
];

describe("ticket-finished default", () => {
  test("fails when the event carries no ticket id, and calls nothing", async () => {
    const ctx = createMockContext(automation.configSchema);

    expect(await handler(finished({ id: "", project_id: "project-1" }), ctx)).toBe(false);
    expect(ctx.calls).toEqual([]);
  });

  test("fails when the event carries no project id, and calls nothing", async () => {
    const ctx = createMockContext(automation.configSchema);

    expect(await handler(finished({ id: "ticket-1", project_id: "" }), ctx)).toBe(false);
    expect(ctx.calls).toEqual([]);
  });

  test("logs and succeeds when the project has no done status", async () => {
    const ctx = createMockContext(automation.configSchema, { responses: { "GET /api/statuses": [{ id: "status-review", kind: "review", position: 1 }] } });

    expect(await handler(finished({ id: "ticket-1", project_id: "project-1" }), ctx)).toBe(true);
    expect(ctx.calls).toEqual([{ method: "GET", path: "/api/statuses", body: undefined }]);
    expect(ctx.logs[0]).toContain("no done status");
  });

  test("treats a status list that is not an array as having no done status", async () => {
    const ctx = createMockContext(automation.configSchema, { responses: { "GET /api/statuses": { message: "odd" } } });

    expect(await handler(finished({ id: "ticket-1", project_id: "project-1" }), ctx)).toBe(true);
    expect(ctx.calls.map((c) => c.method)).toEqual(["GET"]);
  });

  test("lets an API error fail the run", async () => {
    const ctx = createMockContext(automation.configSchema);
    ctx.api.request = async () => {
      throw new ApiError(403, {}, "GET /api/statuses failed: 403");
    };

    await expect(handler(finished({ id: "ticket-1", project_id: "project-1" }), ctx)).rejects.toThrow("403");
  });

  test("moves the ticket to the lowest-position done status of its project", async () => {
    const ctx = createMockContext(automation.configSchema, { responses: { "GET /api/statuses": statuses } });

    expect(await handler(finished({ id: "ticket-1", project_id: "project-1" }), ctx)).toBe(true);
    expect(ctx.calls).toEqual([
      { method: "GET", path: "/api/statuses", body: undefined },
      { method: "PATCH", path: "/api/tickets/ticket-1/status", body: { status: "status-done" } },
    ]);
  });
});
