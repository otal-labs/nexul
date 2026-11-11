import { describe, expect, test } from "bun:test";
import { ApiError } from "@otal-labs/nexul-sdk/api-client";
import { createMockContext, eventFixtures } from "@otal-labs/nexul-sdk/testing";
import automation from "./index.ts";

const handler = automation.getHandler("git.pr_opened")!;
const opened = (linked: string[]) => ({ ...eventFixtures["git.pr_opened"], pr: { ...eventFixtures["git.pr_opened"].pr, linked_ticket_ids: linked } });
const statuses = [
  { id: "status-later-review", kind: "review", position: 4 },
  { id: "status-review", kind: "review", position: 2 },
  { id: "status-done", kind: "done", position: 5 },
];

describe("pr-opened default", () => {
  test("succeeds as a no-op when the PR links no ticket", async () => {
    const ctx = createMockContext(automation.configSchema);

    expect(await handler(opened([]), ctx)).toBe(true);
    expect(ctx.calls).toEqual([]);
  });

  test("fails when a linked ticket carries no project id, and moves nothing", async () => {
    const ctx = createMockContext(automation.configSchema, { responses: { "GET /api/tickets/ticket-1": { id: "ticket-1" } } });

    expect(await handler(opened(["ticket-1"]), ctx)).toBe(false);
    expect(ctx.calls.map((c) => c.method)).toEqual(["GET"]);
  });

  test("logs and succeeds when the project has no review status", async () => {
    const ctx = createMockContext(automation.configSchema, {
      responses: { "GET /api/tickets/ticket-1": { id: "ticket-1", project_id: "project-1" }, "GET /api/statuses": [{ id: "status-done", kind: "done", position: 1 }] },
    });

    expect(await handler(opened(["ticket-1"]), ctx)).toBe(true);
    expect(ctx.calls.map((c) => c.method)).toEqual(["GET", "GET"]);
    expect(ctx.logs.some((l) => l.includes("no review status"))).toBe(true);
  });

  test("lets an API error fail the run", async () => {
    const ctx = createMockContext(automation.configSchema);
    ctx.api.request = async () => {
      throw new ApiError(404, {}, "GET /api/tickets/ticket-1 failed: 404");
    };

    await expect(handler(opened(["ticket-1"]), ctx)).rejects.toThrow("404");
  });

  test("moves every linked ticket to the lowest-position review status, reading each project's statuses once", async () => {
    const ctx = createMockContext(automation.configSchema, {
      responses: {
        "GET /api/tickets/ticket-1": { id: "ticket-1", project_id: "project-1" },
        "GET /api/tickets/ticket-2": { id: "ticket-2", project_id: "project-1" },
        "GET /api/statuses": statuses,
      },
    });

    expect(await handler(opened(["ticket-1", "ticket-2"]), ctx)).toBe(true);
    expect(ctx.calls).toEqual([
      { method: "GET", path: "/api/tickets/ticket-1", body: undefined },
      { method: "GET", path: "/api/statuses", body: undefined },
      { method: "PATCH", path: "/api/tickets/ticket-1/status", body: { status: "status-review" } },
      { method: "GET", path: "/api/tickets/ticket-2", body: undefined },
      { method: "PATCH", path: "/api/tickets/ticket-2/status", body: { status: "status-review" } },
    ]);
  });
});
