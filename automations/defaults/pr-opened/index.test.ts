import { describe, expect, test } from "bun:test";
import { createMockContext, eventFixtures } from "@nexul/sdk/testing";
import automation from "./index.ts";

describe("pr-opened default", () => {
  test("moves every linked ticket to the configured in-review status", async () => {
    const ctx = createMockContext(automation.configSchema, { config: { inReviewStatusId: "status-review" } });
    const payload = { ...eventFixtures["git.pr_opened"], pr: { linked_ticket_ids: ["ticket-1", "ticket-2"] } };

    const result = await automation.getHandler("git.pr_opened")!(payload, ctx);

    expect(result).toBe(true);
    expect(ctx.calls).toEqual([
      { method: "PATCH", path: "/api/tickets/ticket-1/status", body: { status: "status-review" } },
      { method: "PATCH", path: "/api/tickets/ticket-2/status", body: { status: "status-review" } },
    ]);
  });

  test("succeeds as a no-op when the PR links no ticket", async () => {
    const ctx = createMockContext(automation.configSchema, { config: { inReviewStatusId: "status-review" } });
    const payload = { ...eventFixtures["git.pr_opened"], pr: { linked_ticket_ids: [] } };

    const result = await automation.getHandler("git.pr_opened")!(payload, ctx);

    expect(result).toBe(true);
    expect(ctx.calls).toEqual([]);
  });

  test("fails without a configured in-review status when there is work to do", async () => {
    const ctx = createMockContext(automation.configSchema);
    const payload = { ...eventFixtures["git.pr_opened"], pr: { linked_ticket_ids: ["ticket-1"] } };

    const result = await automation.getHandler("git.pr_opened")!(payload, ctx);

    expect(result).toBe(false);
    expect(ctx.calls).toEqual([]);
  });
});
