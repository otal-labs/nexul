import { describe, expect, it } from "vitest";

import { botwebhookFollower } from "@/hooks/BotwebhookHooks";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the bot follower", () => {
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
