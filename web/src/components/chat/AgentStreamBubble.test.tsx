import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AgentStreamBubble } from "@/components/chat/AgentStreamBubble";
import type { AgentStreamFrame } from "@/stores/agentStreamStore";

const frame = (overrides: Partial<AgentStreamFrame>): AgentStreamFrame => ({
  messageId: "m1",
  text: "The edits saved.",
  streaming: true,
  activity: "",
  startedAt: Date.now(),
  handoffs: [],
  ...overrides,
});

describe("AgentStreamBubble", () => {
  it("names the step the Agent is running", () => {
    render(<AgentStreamBubble frame={frame({ activity: "doc_update", activityKind: "tool_call", activityTool: "nexul · doc_update" })} onInterrupt={() => {}} live={null} />);
    expect(screen.getByText(/^Calling /)).toBeInTheDocument();
  });

  it("says the Agent is thinking once its last step has finished", () => {
    render(<AgentStreamBubble frame={frame({ activity: "doc_update", activityKind: "tool_result", activityTool: "nexul · doc_update" })} onInterrupt={() => {}} live={null} />);
    expect(screen.getByText("Thinking")).toBeInTheDocument();
    expect(screen.queryByText(/^Called /)).not.toBeInTheDocument();
  });
});
