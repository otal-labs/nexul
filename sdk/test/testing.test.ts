import { describe, expect, it } from "vitest";
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

  it("exposes a fixture for every generated topic", () => {
    expect(eventFixtures["ticket.created"].ticket.id).toBe("fixture-id");
  });
});
