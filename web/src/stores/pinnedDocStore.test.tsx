import { beforeEach, describe, expect, it } from "vitest";

import { usePinnedDocStore } from "@/stores/pinnedDocStore";

beforeEach(() => usePinnedDocStore.setState({ pinned: {} }));

describe("usePinnedDocStore", () => {
  it("puts the newest pin first, keeps workspaces apart, and unpins on a second toggle", () => {
    const { togglePin } = usePinnedDocStore.getState();
    togglePin("ws-1", "a");
    togglePin("ws-1", "b");
    togglePin("ws-2", "c");
    expect(usePinnedDocStore.getState().pinned).toEqual({ "ws-1": ["b", "a"], "ws-2": ["c"] });

    togglePin("ws-1", "b");
    expect(usePinnedDocStore.getState().pinned["ws-1"]).toEqual(["a"]);
  });
});
