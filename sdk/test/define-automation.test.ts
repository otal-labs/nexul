import { describe, expect, it } from "vitest";
import { defineAutomation } from "../src/define-automation.ts";

describe("defineAutomation", () => {
  it("rejects an empty name", () => {
    expect(() => defineAutomation({ name: "", description: "x" })).toThrow(/non-empty name/);
  });

  it("tracks subscriptions in registration order and dispatches by topic", () => {
    const automation = defineAutomation({ name: "board-pair", description: "moves tickets" });
    const handler = async () => true;
    automation.on("ticket.created", handler);
    automation.on("ticket.finished", handler);

    expect(automation.subscriptions).toEqual(["ticket.created", "ticket.finished"]);
    expect(automation.getHandler("ticket.created")).toBe(handler);
    expect(automation.getHandler("doc.created")).toBeUndefined();
  });

  it("re-registering the same topic replaces the handler, not appends", () => {
    const automation = defineAutomation({ name: "x", description: "y" });
    const first = async () => true;
    const second = async () => false;
    automation.on("ticket.created", first);
    automation.on("ticket.created", second);

    expect(automation.subscriptions).toEqual(["ticket.created"]);
    expect(automation.getHandler("ticket.created")).toBe(second);
  });
});
