import { beforeEach, describe, expect, it } from "vitest";

import { useVoiceStageViewStore } from "@/stores/voiceStageViewStore";

describe("voiceStageViewStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useVoiceStageViewStore.setState({ view: "grid" });
  });

  it("keeps the chosen view across a reload", async () => {
    useVoiceStageViewStore.getState().toggle();
    expect(JSON.parse(localStorage.getItem("voice-stage-view") ?? "{}").state).toEqual({ view: "speaker" });

    useVoiceStageViewStore.setState({ view: "grid" });
    localStorage.setItem("voice-stage-view", JSON.stringify({ state: { view: "speaker" }, version: 0 }));
    await useVoiceStageViewStore.persist.rehydrate();

    expect(useVoiceStageViewStore.getState().view).toBe("speaker");
  });
});
