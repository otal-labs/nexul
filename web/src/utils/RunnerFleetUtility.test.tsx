// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Machine, OpenEnrollment, Runner } from "@/models/Runner";
import { fleetCounts, groupFleet, queueReason, splitTail } from "@/utils/RunnerFleetUtility";

const runner = (name: string, machine: string | undefined, connected: boolean, busy = false): Runner => ({
  id: name,
  name,
  connected,
  last_seen: "2026-10-29T10:00:00Z",
  running_job: busy ? { id: `job-${name}`, kind: "deploy", service: "Atlas API" } : null,
  version: "0.1.88",
  ...(machine && { machine }),
});

const machine = (name: string): Machine => ({ id: name, name, stack_root: "/data/nexul", first_seen: "", last_seen: "", actions: [] });
const code = (name: string, on?: string): OpenEnrollment => ({ id: name, name, created_at: "", expires_at: "", ...(on && { machine: on }) });

describe("queueReason", () => {
  it.each([
    { name: "no runner on the target", runners: [runner("a", "lab", false)], want: "no runner online" },
    { name: "only another machine's runner is up", runners: [runner("a", "prod", true)], want: "no runner online" },
    { name: "every connected runner there is busy", runners: [runner("a", "lab", true, true), runner("b", "lab", false)], want: "all runners busy" },
    { name: "one runner there is idle", runners: [runner("a", "lab", true, true), runner("b", "lab", true)], want: "next in line" },
  ])("$name", ({ runners, want }) => {
    expect(queueReason({ target: "lab" }, runners)).toBe(want);
  });

  it("lets a job for no machine wait on any runner", () => {
    expect(queueReason({}, [runner("a", "prod", true, true)])).toBe("all runners busy");
    expect(queueReason({ target: "" }, [])).toBe("no runner online");
  });
});

describe("groupFleet", () => {
  it("puts each runner and code under its machine and the rest in a last group", () => {
    const groups = groupFleet(
      [machine("prod"), machine("spare")],
      [runner("a", "prod", true), runner("legacy", undefined, false), runner("ghost", "gone", false)],
      [code("next", "prod"), code("laptop")],
    );

    expect(groups.map((g) => [g.machine?.name, g.runners.map((r) => r.name), g.codes.map((c) => c.name)])).toEqual([
      ["prod", ["a"], ["next"]],
      ["spare", [], []],
      [undefined, ["legacy", "ghost"], ["laptop"]],
    ]);
  });

  it("adds no last group when every row has a machine", () => {
    expect(groupFleet([machine("prod")], [runner("a", "prod", true)], [])).toHaveLength(1);
  });
});

describe("fleetCounts", () => {
  it("splits runners into online, busy and offline", () => {
    expect(fleetCounts([runner("a", "p", true, true), runner("b", "p", true), runner("c", "p", false)])).toEqual({
      total: 3,
      online: 2,
      busy: 1,
      offline: 1,
    });
  });
});

describe("splitTail", () => {
  it.each([
    ["build-farm-eu-central-07-node-14", ["build-farm-eu-central-07-", "node-14"]],
    ["worker-06-decommissioned-soon", ["worker-06-decommissioned-", "soon"]],
    ["worker-02", ["worker-", "02"]],
    ["legacy", ["legacy", ""]],
    ["-leading", ["-leading", ""]],
    ["runner-averyveryverylongtail", ["runner-averyveryverylongtail", ""]],
  ])("%s", (name, want) => {
    expect(splitTail(name)).toEqual(want);
  });
});
