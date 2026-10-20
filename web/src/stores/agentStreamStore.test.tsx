// @vitest-environment node
import { beforeEach, describe, expect, it } from "vitest";

import { useAgentStreamStore } from "@/stores/agentStreamStore";

describe("agentStreamStore", () => {
  beforeEach(() => {
    useAgentStreamStore.setState({ streams: {} });
  });

  it("setStream upserts a conversation's frame, replacing the previous text", () => {
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "Thinking", streaming: true });
    useAgentStreamStore.getState().setStream("c1", { messageId: "m1", text: "Thinking about it", streaming: true });

    expect(useAgentStreamStore.getState().streams.c1).toMatchObject({ messageId: "m1", text: "Thinking about it", streaming: true });
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

describe("agentStreamStore steps", () => {
  beforeEach(() => {
    useAgentStreamStore.setState({ streams: {} });
  });

  const said = (conversationId: string) =>
    (useAgentStreamStore.getState().streams[conversationId]?.steps ?? []).map((s) => (s.kind === "text" ? s.detail : `${s.kind} ${s.call_id ?? ""}`));

  it("keeps each message's text and each tool step, a result replacing its call", () => {
    const { setStream } = useAgentStreamStore.getState();
    setStream("c1", { messageId: "m1", text: "Checking the doc.", streaming: true });
    setStream("c1", { messageId: "m1", text: "Checking the doc.", streaming: true, activity: "doc_get", activityKind: "tool_call", activityTool: "nexul · doc_get", activityCallId: "t1" });
    setStream("c1", { messageId: "m1", text: "Checking the doc.", streaming: true, activity: "doc_get", activityKind: "tool_result", activityTool: "nexul · doc_get", activityCallId: "t1" });
    setStream("c1", { messageId: "m2", text: "Master has", streaming: true });
    setStream("c1", { messageId: "m2", text: "Master hasn't moved.", streaming: true });
    setStream("c1", { messageId: "m2", text: "Master hasn't moved.", streaming: true, activity: "git", activityKind: "tool_call", activityTool: "Bash", activityCallId: "t2" });

    expect(said("c1")).toEqual(["Checking the doc.", "tool_result t1", "Master hasn't moved.", "tool_call t2"]);
  });

  it("splits one message's text at a tool call, so the text after it is a step of its own", () => {
    const { setStream } = useAgentStreamStore.getState();
    setStream("c1", { messageId: "m1", text: "Reading.", streaming: true });
    setStream("c1", { messageId: "m1", text: "Reading.", streaming: true, activity: "Read", activityKind: "tool_call", activityTool: "Read", activityCallId: "t1" });
    setStream("c1", { messageId: "m1", text: "Reading. Now editing.", streaming: true });

    expect(said("c1")).toEqual(["Reading.", "tool_call t1", "Now editing."]);
  });
});
