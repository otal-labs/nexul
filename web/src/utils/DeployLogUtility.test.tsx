import { describe, expect, it } from "vitest";

import { DeployStatus, DeployStrategy, type Deploy, type DeployLogLine } from "@/models/Stack";
import { deriveDeployProgress, formatLogTimestamp, formatStepDuration, logToText } from "@/utils/DeployLogUtility";

const T0 = Date.parse("2026-09-22T10:00:00Z");
const at = (s: number) => T0 + s * 1000;
const iso = (s: number) => new Date(at(s)).toISOString();

const deploy = (overrides: Partial<Deploy>): Deploy => ({
  id: "d-1",
  kind: "build",
  stack_id: "stack-1",
  service: "api",
  target: "instance",
  image: "",
  status: DeployStatus.Running,
  strategy: DeployStrategy.Compose,
  created_at: iso(0),
  updated_at: iso(0),
  ...overrides,
});

const line = (seq: number, s: number, phase: DeployLogLine["phase"], text = `line ${seq}`): DeployLogLine => ({
  seq,
  ts: at(s),
  phase,
  text,
});

const stateOf = (deploy: Deploy, lines: DeployLogLine[], now: number) =>
  deriveDeployProgress(deploy, lines, now).steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)]);

describe("deriveDeployProgress", () => {
  it("shows a pending build as waiting with every phase still to come", () => {
    const progress = deriveDeployProgress(deploy({ status: DeployStatus.Pending }), [], at(4));
    expect(progress.title).toBe("Building and deploying");
    expect(progress.steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)])).toEqual([
      ["wait", "active", "4s"],
      ["checkout", "pending", "—"],
      ["build", "pending", "—"],
      ["deploy", "pending", "—"],
    ]);
  });

  it("closes each phase at the next phase's first line and ticks the active one from now", () => {
    const lines = [line(1, 5, "checkout"), line(2, 9, "checkout"), line(3, 12, "build")];
    expect(stateOf(deploy({}), lines, at(30))).toEqual([
      ["wait", "done", "5s"],
      ["checkout", "done", "7s"],
      ["build", "active", "18s"],
      ["deploy", "pending", "—"],
    ]);
  });

  it("ends the last started phase at updated_at once the deploy is healthy", () => {
    const lines = [line(1, 5, "checkout"), line(2, 10, "build"), line(3, 80, "deploy")];
    const progress = deriveDeployProgress(deploy({ status: DeployStatus.Healthy, updated_at: iso(95) }), lines, at(9999));
    expect(progress.title).toBe("Deployed");
    expect(progress.steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)])).toEqual([
      ["wait", "done", "5s"],
      ["checkout", "done", "5s"],
      ["build", "done", "1m 10s"],
      ["deploy", "done", "15s"],
    ]);
  });

  it("marks the last started step failed and leaves unstarted phases skipped", () => {
    const lines = [line(1, 5, "checkout"), line(2, 10, "build"), line(3, 20, "", "deploy failed: exit 1")];
    const progress = deriveDeployProgress(deploy({ status: DeployStatus.Failed, updated_at: iso(20) }), lines, at(9999));
    expect(progress.title).toBe("Deploy failed");
    expect(progress.steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)])).toEqual([
      ["wait", "done", "5s"],
      ["checkout", "done", "5s"],
      ["build", "failed", "10s"],
      ["deploy", "skipped", "—"],
    ]);
  });

  it("treats a phase with no lines as skipped once a later phase has started", () => {
    const lines = [line(1, 5, "build"), line(2, 30, "deploy")];
    expect(stateOf(deploy({}), lines, at(40))).toEqual([
      ["wait", "done", "5s"],
      ["checkout", "skipped", "—"],
      ["build", "done", "25s"],
      ["deploy", "active", "10s"],
    ]);
  });

  it("fails the waiting step when only a phase-less line arrived before the deploy failed", () => {
    const lines = [line(1, 3, "", "deploy failed: no runner")];
    expect(stateOf(deploy({ status: DeployStatus.Failed, updated_at: iso(3) }), lines, at(9999))).toEqual([
      ["wait", "failed", "3s"],
      ["checkout", "skipped", "—"],
      ["build", "skipped", "—"],
      ["deploy", "skipped", "—"],
    ]);
  });

  it("derives two steps for an image-only deploy", () => {
    const lines = [line(1, 2, "deploy")];
    const progress = deriveDeployProgress(deploy({ kind: "deploy", image: "ghcr.io/onik/api:v1" }), lines, at(10));
    expect(progress.title).toBe("Deploying");
    expect(progress.steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)])).toEqual([
      ["wait", "done", "2s"],
      ["deploy", "active", "8s"],
    ]);
  });

  it("treats a missing kind as an image-only deploy", () => {
    const kindless = deploy({ status: DeployStatus.Healthy, updated_at: iso(7) });
    delete kindless.kind;
    const progress = deriveDeployProgress(kindless, [], at(99));
    expect(progress.steps.map((s) => [s.key, s.state, formatStepDuration(s.durationMs)])).toEqual([
      ["wait", "done", "7s"],
      ["deploy", "skipped", "—"],
    ]);
  });
});

describe("formatStepDuration", () => {
  it("rolls seconds into minutes and hours", () => {
    expect(formatStepDuration(0)).toBe("0s");
    expect(formatStepDuration(59_999)).toBe("59s");
    expect(formatStepDuration(72_000)).toBe("1m 12s");
    expect(formatStepDuration(3_723_000)).toBe("1h 2m 3s");
  });
});

describe("log text", () => {
  it("formats a local timestamp with millisecond precision", () => {
    const ts = new Date(2026, 8, 22, 14, 37, 8, 112).getTime();
    expect(formatLogTimestamp(ts)).toBe("14:37:08.112");
  });

  it("joins lines as timestamp, two spaces, text", () => {
    const ts = new Date(2026, 8, 22, 14, 37, 8, 5).getTime();
    expect(logToText([{ seq: 1, ts, phase: "build", text: "$ bun install" }, { seq: 2, ts, phase: "build", text: "done" }])).toBe(
      "14:37:08.005  $ bun install\n14:37:08.005  done",
    );
  });
});
