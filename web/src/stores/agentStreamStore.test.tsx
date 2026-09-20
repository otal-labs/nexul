import { beforeEach, describe, expect, it } from "vitest";

import { useAgentStreamStore } from "@/stores/agentStreamStore";

describe("agentStreamStore", () => {
  beforeEach(() => {
    useAgentStreamStore.setState({ streams: {} });
  });

  it("starts empty", () => {
    expect(useAgentStreamStore.getState().streams).toEqual({});
  });

  it("setStream upserts a conversation's frame, replacing the previous text", () => {
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "Thinking", streaming: true });
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "Thinking about it", streaming: true });

    expect(useAgentStreamStore.getState().streams.c1).toMatchObject({ messageId: "m1", text: "Thinking about it", streaming: true });
  });

  it("keeps separate conversations' frames independent", () => {
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "hi", streaming: true });
    useAgentStreamStore.getState().setStream("c2", { messageId: "m2", text: "hello", streaming: true });

    expect(Object.keys(useAgentStreamStore.getState().streams)).toEqual(["c1", "c2"]);
  });

  it("clearStream removes only the given conversation's frame", () => {
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "hi", streaming: false });
    useAgentStreamStore.getState().setStream("c2", { messageId: "m2", text: "hello", streaming: false });

    useAgentStreamStore.getState().clearStream("c1");

    expect(useAgentStreamStore.getState().streams.c1).toBeUndefined();
    expect(useAgentStreamStore.getState().streams.c2).toBeDefined();
  });

  it("clearStream on an already-absent conversation is a no-op", () => {
    const before = useAgentStreamStore.getState().streams;
    useAgentStreamStore.getState().clearStream("missing");
    expect(useAgentStreamStore.getState().streams).toBe(before);
  });
});

describe("agentStreamStore startedAt", () => {
  it("is set on the first frame and preserved across replaces", () => {
    useAgentStreamStore.setState({ streams: {} });
    useAgentStreamStore.getState().setStream("c1", { messageId: "", text: "", streaming: true });
    const first = useAgentStreamStore.getState().streams.c1?.startedAt;
    expect(first).toBeGreaterThan(0);
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "more", streaming: true });
    expect(useAgentStreamStore.getState().streams.c1?.startedAt).toBe(first);
  });
});
