import { describe, expect, it } from "vitest";

import { reactionFollower } from "@/hooks/ReactionHooks";
import type { Message } from "@/models/Chat";
import { followFrame, seeded } from "@/test/followFrame";

describe("the reaction follower", () => {
  it("adds and removes someone's reaction on the cached message", async () => {
    const posted = { id: "m-1", conversation_id: "c-1", author_id: "u-1", author_kind: "user", body: "ship it", mentions: null, created_at: "", updated_at: "" };
    const client = seeded([[["getChatMessages", "c-1", undefined], [posted]]]);
    const react = (reacted: boolean) =>
      followFrame(reactionFollower, "chat.message.reactions_changed", { conversation_id: "c-1", message_id: "m-1", user_id: "u-2", emoji: "🚀", reacted }, client);
    await react(true);
    expect(client.getQueryData<Message[]>(["getChatMessages", "c-1", undefined])?.[0]?.reactions).toEqual([{ emoji: "🚀", user_ids: ["u-2"] }]);
    await react(false);
    expect(client.getQueryData<Message[]>(["getChatMessages", "c-1", undefined])?.[0]?.reactions ?? []).toEqual([]);
  });
});
