import { describe, expect, it } from "bun:test";
import type { QueuedRun } from "../src/context.ts";
import { createMockContext } from "../src/testing.ts";
import { eventFixtures } from "../src/events.generated.ts";

describe("createMockContext", () => {
  it("records API calls instead of making them", async () => {
    const ctx = createMockContext({});
    await ctx.api.automations.setEnabled("auto-1", true);

    expect(ctx.calls).toEqual([{ method: "PATCH", path: "/api/automations/auto-1/enabled", body: { enabled: true } }]);
  });

  it("returns scripted responses without ever hitting a real network", async () => {
    const ctx = createMockContext(
      {},
      { responses: { "GET /api/automations/auto-1": { id: "auto-1", name: "board-pair" } } },
    );
    const result = await ctx.api.automations.get("auto-1");
    expect(result).toEqual({ id: "auto-1", name: "board-pair" });
  });

  it("seeds config from the schema's defaults, overridable per test", () => {
    const ctx = createMockContext({ threshold: { type: "number", default: 3 } }, { config: { threshold: 9 } });
    expect(ctx.config.threshold).toBe(9);
  });

  it("captures ctx.log calls", () => {
    const ctx = createMockContext({});
    ctx.log("hello", { id: 1 });
    expect(ctx.logs).toEqual([JSON.stringify({ message: "hello", id: 1 })]);
  });

  it("records runPlay as the queue call it makes, answering with a queued run", async () => {
    const ctx = createMockContext({});
    const run = await ctx.runPlay("Fix with AI", "ticket-1", { runOn: "tester", priority: "high" });

    expect(run).toEqual({ id: "mock-run", state: "queued", reason: "" });
    expect(ctx.calls).toEqual([
      { method: "POST", path: "/api/plays/queue", body: { play: "Fix with AI", ticket_id: "ticket-1", run_on: "tester", priority: "high" } },
    ]);
  });

  it("lets a test script what runPlay answers", async () => {
    const didntRun: QueuedRun = { id: "q-1", state: "didnt_run", reason: "nobody to run it on: the ticket has no developer" };
    const ctx = createMockContext({}, { responses: { "POST /api/plays/queue": didntRun } });
    expect(await ctx.runPlay("Fix with AI", "ticket-1")).toEqual(didntRun);
    expect(ctx.calls[0]?.body).toEqual({ play: "Fix with AI", ticket_id: "ticket-1" });
  });

  it("exposes a fixture for every generated topic", () => {
    expect(eventFixtures["ticket.created"].ticket.id).toBe("fixture-id");
  });
});
