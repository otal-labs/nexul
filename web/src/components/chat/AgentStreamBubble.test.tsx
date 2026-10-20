import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AgentStreamBubble } from "@/components/chat/AgentStreamBubble";
import type { ActivityEntry } from "@/models/Trail";
import type { AgentStreamFrame } from "@/stores/agentStreamStore";

const frame = (overrides: Partial<AgentStreamFrame>): AgentStreamFrame => ({
  messageId: "m1",
  text: "",
  streaming: true,
  activity: "",
  startedAt: Date.now(),
  handoffs: [],
  steps: [],
  textFrom: 0,
  ...overrides,
});

const at = new Date().toISOString();
const said = (text: string): ActivityEntry => ({ kind: "text", summary: text, detail: text, at });
const docUpdate = (kind: "tool_call" | "tool_result"): ActivityEntry => ({ kind, call_id: "t1", tool: "nexul · doc_update", summary: "doc_update", at });

describe("AgentStreamBubble", () => {
  it("names the step the Agent is running", () => {
    render(<AgentStreamBubble frame={frame({ steps: [docUpdate("tool_call")] })} onInterrupt={() => {}} live={null} />);
    expect(screen.getAllByText(/^Calling /).length).toBeGreaterThan(0);
  });

  it("says the Agent is thinking once its last step has finished", () => {
    render(<AgentStreamBubble frame={frame({ steps: [docUpdate("tool_result")] })} onInterrupt={() => {}} live={null} />);
    expect(screen.getByText("Thinking")).toBeInTheDocument();
  });

  it("says the Agent is thinking before its first step or text", () => {
    render(<AgentStreamBubble frame={frame({})} onInterrupt={() => {}} live={null} />);
    expect(screen.getByText("Thinking")).toBeInTheDocument();
  });

  it("keeps what the Agent said before a step under what it says after", () => {
    const steps = [said("Express is confirmed."), docUpdate("tool_result"), said("Master hasn't moved.")];
    render(<AgentStreamBubble frame={frame({ text: "Master hasn't moved.", steps })} onInterrupt={() => {}} live={null} />);
    expect(screen.getByText("Express is confirmed.")).toBeInTheDocument();
    expect(screen.getByText("Master hasn't moved.")).toBeInTheDocument();
  });
});
