import { describe, expect, it } from "vitest";

import { computerSetupFollower } from "@/hooks/ComputerSetupHooks";
import { useSetupActivityStore } from "@/stores/setupActivityStore";
import { followFrame, isStale, seeded } from "@/test/followFrame";

describe("the computer setup follower", () => {
  it("refetches the setup of the computer whose turn moved, and no other", async () => {
    const client = seeded([
      [["getComputerSetup", "c1"], {}],
      [["getComputerSetup", "c2"], {}],
    ]);
    await followFrame(computerSetupFollower, "computer.setup_turn_changed", { computer_id: "c2", user_id: "u1", state: "running" }, client);
    expect([isStale(client, ["getComputerSetup", "c1"]), isStale(client, ["getComputerSetup", "c2"])]).toEqual([false, true]);
  });

  it("appends a setup turn's steps to the activity store without a request, one row per tool call, a message whole", async () => {
    useSetupActivityStore.setState({ steps: {} });
    const client = seeded([[["getComputerSetup", "c1"], {}]]);
    const activity = (extra: Record<string, string>) =>
      followFrame(computerSetupFollower, "computer.setup_turn_activity", { computer_id: "c1", turn_id: "t1", provider: "codex", ...extra }, client);
    await activity({ status: "nexul mcp add", kind: "tool_call", call_id: "call-1", tool: "Shell", at: "2026-10-01T10:00:00Z" });
    await activity({ status: "nexul mcp add", kind: "tool_result", call_id: "call-1", tool: "Shell", at: "2026-10-01T10:00:02Z" });
    await activity({ status: "All set. Codex…", kind: "text", text: "All set. Codex is confirmed.", at: "2026-10-01T10:00:05Z" });
    expect(useSetupActivityStore.getState().steps.t1).toEqual([
      { kind: "tool_result", call_id: "call-1", tool: "Shell", summary: "nexul mcp add", detail: "", at: "2026-10-01T10:00:02Z" },
      { kind: "text", call_id: "", tool: "", summary: "All set. Codex…", detail: "All set. Codex is confirmed.", at: "2026-10-01T10:00:05Z" },
    ]);
    expect(isStale(client, ["getComputerSetup", "c1"])).toBe(false);
  });
});
