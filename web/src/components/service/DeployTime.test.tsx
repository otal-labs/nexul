import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatDayLabel, formatRelativeTime, groupDeploysByDay } from "@/components/service/DeployTime";
import { DeployStatus, DeployStrategy, type Deploy } from "@/models/Stack";

const NOW = "2026-08-12T12:00:00Z";

describe("formatRelativeTime", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(NOW));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it.each([
    ["", ""],
    ["garbage", "garbage"],
    ["2026-08-12T12:00:00Z", "just now"],
    ["2026-08-12T11:59:30Z", "just now"],
    ["2026-08-12T11:30:00Z", "30m ago"],
    ["2026-08-12T09:00:00Z", "3h ago"],
    ["2026-08-10T12:00:00Z", "2d ago"],
    ["2026-07-14T12:00:00Z", "29d ago"],
    ["2026-07-13T12:00:00Z", "1mo ago"],
    ["2026-01-12T12:00:00Z", "7mo ago"],
    ["2025-08-12T12:00:00Z", "1y ago"],
  ])("formats %s", (input, expected) => {
    expect(formatRelativeTime(input)).toBe(expected);
  });
});

const deploy = (overrides: Partial<Deploy>): Deploy => ({
  id: "d-0",
  stack_id: "stack-1",
  service: "api",
  target: "10.0.0.1:22",
  image: "ghcr.io/onik/api:v1",
  status: DeployStatus.Healthy,
  strategy: DeployStrategy.Compose,
  log: "",
  created_at: NOW,
  updated_at: NOW,
  ...overrides,
});

describe("formatDayLabel", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(NOW));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it.each([
    ["", "Unknown date"],
    ["garbage", "Unknown date"],
    ["2026-08-12T00:00:00Z", "Today"],
    ["2026-08-12T23:59:59Z", "Today"],
    ["2026-08-11T23:59:59Z", "Yesterday"],
    ["2026-08-01T09:00:00Z", "Aug 1"],
    ["2025-12-31T09:00:00Z", "Dec 31, 2025"],
  ])("labels %s as %s", (input, expected) => {
    expect(formatDayLabel(input)).toBe(expected);
  });
});

describe("groupDeploysByDay", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(NOW));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("folds consecutive same-day deploys into one group", () => {
    const deploys = [
      deploy({ id: "d-1", created_at: "2026-08-12T10:00:00Z" }),
      deploy({ id: "d-2", created_at: "2026-08-12T09:00:00Z" }),
    ];
    const groups = groupDeploysByDay(deploys);
    expect(groups).toHaveLength(1);
    expect(groups[0]?.label).toBe("Today");
    expect(groups[0]?.deploys.map((d) => d.id)).toEqual(["d-1", "d-2"]);
  });

  it("splits deploys from different days into separate groups, preserving order", () => {
    const deploys = [
      deploy({ id: "d-1", created_at: "2026-08-12T10:00:00Z" }),
      deploy({ id: "d-2", created_at: "2026-08-11T09:00:00Z" }),
      deploy({ id: "d-3", created_at: "2026-08-01T09:00:00Z" }),
    ];
    const groups = groupDeploysByDay(deploys);
    expect(groups.map((g) => g.label)).toEqual(["Today", "Yesterday", "Aug 1"]);
    expect(groups[0]?.deploys.map((d) => d.id)).toEqual(["d-1"]);
  });

  it("returns no groups for an empty list", () => {
    expect(groupDeploysByDay([])).toEqual([]);
  });
});
