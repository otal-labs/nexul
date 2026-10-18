// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { ErrorRateEpisode, StackWithBranches } from "@/models/Stack";
import { errorRateHealth, errorRateOf, recoveryText, ruleText } from "@/utils/ErrorRateUtility";

const on = { enabled: true, percent: 5, minutes: 5, min_requests: 20 };

const stack = (change: Partial<StackWithBranches> = {}): StackWithBranches => ({
  id: "stack-1",
  project_id: "proj-1",
  name: "api",
  slug: "api",
  machine: "box",
  strategy: "compose",
  managed: true,
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
  ...change,
});

const episode: ErrorRateEpisode = {
  id: "ep-1",
  stack_id: "stack-1",
  started_at: "2026-10-16T09:42:00Z",
  peak_percent: 7.24,
  requests: 400,
  errors: 29,
  limit_percent: 5,
  window_minutes: 5,
  min_requests: 20,
};

describe("errorRateOf", () => {
  it("starts a stack with no rule off at the defaults", () => {
    expect(errorRateOf(stack())).toEqual({ enabled: false, percent: 5, minutes: 5, min_requests: 20 });
  });
});

describe("the rule in words", () => {
  it("states what trips it and what ends it", () => {
    expect(ruleText(on)).toBe("5xx above 5% for 5 minutes, with at least 20 requests");
    expect(recoveryText(on)).toBe("Recovers after 5 minutes at or under 5%.");
  });

  it("reads one minute, a fractional limit and a large floor", () => {
    const rule = { enabled: true, percent: 2.5, minutes: 1, min_requests: 12000 };
    expect(ruleText(rule)).toBe("5xx above 2.5% for 1 minute, with at least 12,000 requests");
    expect(recoveryText(rule)).toBe("Recovers after 1 minute at or under 2.5%.");
  });
});

describe("errorRateHealth", () => {
  it("is the warning line for an open episode, at its peak", () => {
    expect(errorRateHealth(stack({ error_rate: on, error_rate_episode: episode }))).toEqual({ tone: "warning", text: "5xx 7.2%", at: episode.started_at });
  });

  it("is the success line for an episode that ended within the hour", () => {
    expect(errorRateHealth(stack({ error_rate: on, error_rate_recovered_at: "2026-10-16T09:52:00Z" }))).toEqual({
      tone: "success",
      text: "5xx recovered",
      at: "2026-10-16T09:52:00Z",
    });
  });

  it("is nothing when the stack is fine, and nothing at all while the rule is off", () => {
    expect(errorRateHealth(stack({ error_rate: on }))).toBeUndefined();
    expect(errorRateHealth(stack({ error_rate: { ...on, enabled: false }, error_rate_episode: episode }))).toBeUndefined();
    expect(errorRateHealth(stack({ error_rate_episode: episode }))).toBeUndefined();
  });
});
