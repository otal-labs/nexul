import { describe, expect, it } from "vitest";

import type { ServiceOutage } from "@/models/Stack";
import { outageText } from "@/utils/OutageUtility";

const outage = (over: Partial<ServiceOutage> = {}): ServiceOutage => ({
  id: "out-1",
  service: "web",
  container: "shop-web-1",
  started_at: "2026-10-01T10:42:00Z",
  exit_code: 1,
  oom: false,
  restart_count: 0,
  ...over,
});

const clock = /\d\d:\d\d/;

describe("outageText", () => {
  it.each([
    ["a crash", outage(), /^Down since \d\d:\d\d · exit 1$/],
    ["an out-of-memory kill", outage({ exit_code: 137, oom: true }), /^Down since \d\d:\d\d · exit 137 · out of memory$/],
    ["an OOM kill whose death has not been seen", outage({ exit_code: 0, oom: true }), /^Down since \d\d:\d\d · out of memory$/],
    ["a restart loop", outage({ restart_count: 3 }), /^Down since \d\d:\d\d · exit 1 · crashed 3 more times$/],
    ["a second crash", outage({ restart_count: 1 }), /^Down since \d\d:\d\d · exit 1 · crashed 1 more time$/],
  ])("%s", (_, o, want) => {
    expect(outageText(o)).toMatch(want);
  });

  it("keeps the raw time when it cannot be read", () => {
    expect(outageText(outage({ started_at: "never" }))).toBe("Down since never · exit 1");
    expect(clock.test(outageText(outage()))).toBe(true);
  });
});
