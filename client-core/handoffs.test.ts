import { describe, expect, it } from "vitest";

import { handoffsSpanSeconds, handoffsStartMs, handoffTally, subAgentCount, type HandoffSummary } from "@nexul/client-core/handoffs";

const at = (s: number) => `2026-10-04T10:00:${String(s).padStart(2, "0")}Z`;
const helper = (state: HandoffSummary["state"], ...times: number[]): HandoffSummary => ({
  state,
  steps: times.map((s) => ({ kind: "tool_result", tool: "Bash", summary: "go test", at: at(s) })),
});

describe("handoffs", () => {
  it("counts the agents by state, the running ones first and empty states left out", () => {
    expect(handoffTally([helper("done"), helper("running"), helper("failed"), helper("running")])).toBe("2 working, 1 failed, 1 done");
    expect(subAgentCount(1)).toBe("1 sub-agent");
    expect(subAgentCount(3)).toBe("3 sub-agents");
  });

  it("spans from the first agent's first step to the last agent's last, and knows nothing without times", () => {
    const helpers = [helper("done", 5, 20), helper("done", 2, 9), helper("running")];
    expect(handoffsStartMs(helpers)).toBe(new Date(at(2)).getTime());
    expect(handoffsSpanSeconds(helpers)).toBe(18);
    expect(handoffsSpanSeconds([helper("running")])).toBeNull();
  });
});
