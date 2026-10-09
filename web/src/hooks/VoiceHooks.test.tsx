import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { voiceFollower } from "@/hooks/VoiceHooks";
import { useVoiceOccupancyStore } from "@/stores/voiceOccupancyStore";
import { followFrame } from "@/test/followFrame";

describe("the voice follower", () => {
  it("replaces a channel's occupants wholesale, and forgets a channel once it empties", async () => {
    const client = new QueryClient();
    await followFrame(voiceFollower, "voice.occupancy.changed", { conversation_id: "c-1", occupants: [{ identity: "u-2", name: "Lena" }] }, client);
    expect(useVoiceOccupancyStore.getState().occupancy["c-1"]).toEqual([{ identity: "u-2", name: "Lena" }]);
    await followFrame(voiceFollower, "voice.occupancy.changed", { conversation_id: "c-1", occupants: [] }, client);
    expect(useVoiceOccupancyStore.getState().occupancy["c-1"]).toBeUndefined();
  });
});
