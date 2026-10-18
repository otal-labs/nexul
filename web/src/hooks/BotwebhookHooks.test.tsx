// @vitest-environment node
import { describe, expect, it } from "vitest";

import { botwebhookFollower } from "@/hooks/BotwebhookHooks";
import type { Botwebhook } from "@/models/Botwebhook";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the bot follower", () => {
  it("refreshes live bots in the workspace whose role changed, leaving other workspaces and deleted bots alone", async () => {
    const client = seeded([]);
    for (const [conversationId, workspaceId, deleted] of [["c-1", "ws-1", false], ["c-2", "ws-2", false], ["c-1", "ws-1", true]] as const) {
      await client.fetchQuery({
        queryKey: ["getBotwebhooks", conversationId, deleted],
        queryFn: async () => [],
        meta: { workspaceId },
      });
    }
    await followFrame(botwebhookFollower, "role.updated", { role_id: "role-1", workspace_id: "ws-1" }, client);
    expect(isStale(client, ["getBotwebhooks", "c-1", false])).toBe(true);
    expect(isStale(client, ["getBotwebhooks", "c-2", false])).toBe(false);
    expect(isStale(client, ["getBotwebhooks", "c-1", true])).toBe(false);
  });

  it.each([
    ["workspace.member.added", { user_id: "bob", workspace_id: "ws-1" }],
    ["workspace.member.updated", { user_id: "bob", workspace_id: "ws-1" }],
    ["workspace.member.removed", { user_id: "bob", workspace_id: "ws-1" }],
    ["account.disabled", { account_id: "bob" }],
    ["account.reactivated", { account_id: "bob" }],
    ["account.removed", { account_id: "bob" }],
    ["account.restored", { account_id: "bob" }],
  ])("refreshes bots made by the person whose posting changed on %s, leaving other creators' bots alone", async (topic, payload) => {
    const bot = (id: string, conversationId: string, creator: string): Botwebhook => ({
      id, conversation_id: conversationId, created_by: creator, name: "CI", created_at: "", updated_at: "", post_count: 0,
    });
    const client = seeded([]);
    for (const [row, workspaceId, deleted] of [
      [bot("b-1", "c-1", "bob"), "ws-1", false],
      [bot("b-2", "c-2", "alice"), "ws-1", false],
      [{ ...bot("b-old", "c-1", "bob"), deleted_at: "2026-01-01T00:00:00Z" }, "ws-1", true],
      [bot("b-3", "c-3", "bob"), "ws-2", false],
    ] as const) {
      await client.fetchQuery({
        queryKey: ["getBotwebhooks", row.conversation_id, deleted],
        queryFn: async () => [row],
        meta: { workspaceId },
      });
    }
    await followFrame(botwebhookFollower, topic, payload, client);
    expect(isStale(client, ["getBotwebhooks", "c-1", false])).toBe(true);
    expect(isStale(client, ["getBotwebhooks", "c-2", false])).toBe(false);
    expect(isStale(client, ["getBotwebhooks", "c-1", true])).toBe(false);
    expect(isStale(client, ["getBotwebhooks", "c-3", false])).toBe(!("workspace_id" in payload));
  });

  it("refetches the bot lists of the conversation the bot belongs to, deleted ones included, and no other", async () => {
    const client = seeded([
      [["getBotwebhooks", "c-1", false], []],
      [["getBotwebhooks", "c-1", true], []],
      [["getBotwebhooks", "c-2", false], []],
    ]);
    await followFrame(botwebhookFollower, "botwebhook.deleted", { botwebhook_id: "b-1", conversation_id: "c-1", workspace_id: "ws-1", name: "ci" }, client);
    const keys = [["getBotwebhooks", "c-1", false], ["getBotwebhooks", "c-1", true], ["getBotwebhooks", "c-2", false]];
    expect(keys.map((key) => isStale(client, key))).toEqual([true, true, false]);
  });
});
