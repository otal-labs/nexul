import { describe, expect, test } from "bun:test";
import { createMockContext, eventFixtures } from "@otal-labs/nexul-sdk/testing";
import automation from "./index.ts";

const config = { channel: "chan-crashes", projectId: "project-1" };
const handler = automation.getHandler("chat.bot_message.posted")!;

describe("crash-intake default", () => {
  test("hands a post in the configured channel to the intake", async () => {
    const ctx = createMockContext(automation.configSchema, {
      config,
      responses: { "POST /api/crash-reports": { outcome: "filed", ticket_id: "ticket-1" } },
    });
    const embeds = [{ title: "GetSource crash report", fields: [] }];
    const payload = { ...eventFixtures["chat.bot_message.posted"], conversation_id: "chan-crashes", content: "Crash fingerprint: ab", embeds };

    const result = await handler(payload, ctx);

    expect(result).toBe(true);
    expect(ctx.calls).toEqual([
      {
        method: "POST",
        path: "/api/crash-reports",
        body: { project_id: "project-1", bug_type: "bug", label: "high-priority", content: "Crash fingerprint: ab", embeds },
      },
    ]);
    expect(ctx.logs[0]).toContain("crash report filed");
  });

  test("leaves posts in other channels alone", async () => {
    const ctx = createMockContext(automation.configSchema, { config });
    const payload = { ...eventFixtures["chat.bot_message.posted"], conversation_id: "chan-general" };

    expect(await handler(payload, ctx)).toBe(true);
    expect(ctx.calls).toEqual([]);
  });

  test("fails without a configured channel or project, and calls nothing", async () => {
    const ctx = createMockContext(automation.configSchema, { config: { channel: "chan-crashes" } });
    const payload = { ...eventFixtures["chat.bot_message.posted"], conversation_id: "chan-crashes" };

    expect(await handler(payload, ctx)).toBe(false);
    expect(ctx.calls).toEqual([]);
  });

  test("sends an empty list when the post carried no embeds", async () => {
    const ctx = createMockContext(automation.configSchema, {
      config,
      responses: { "POST /api/crash-reports": { outcome: "ignored", reason: "not a crash report" } },
    });
    const { embeds: _, ...payload } = { ...eventFixtures["chat.bot_message.posted"], conversation_id: "chan-crashes" };

    expect(await handler(payload, ctx)).toBe(true);
    expect((ctx.calls[0]?.body as { embeds: unknown }).embeds).toEqual([]);
  });
});
