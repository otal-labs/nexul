// @vitest-environment node
import { describe, expect, it } from "vitest";

import { actorLabel, dayLabel, deltaSign } from "@/utils/HistoryChangeUtility";

describe("actorLabel", () => {
  it("names the person, then what the change came through", () => {
    expect(actorLabel({ kind: "user", user_id: "u-1" }, "Alice")).toBe("Alice");
    expect(actorLabel({ kind: "user:mcp", user_id: "u-1" }, "Alice")).toBe("Alice via an agent");
    expect(actorLabel({ kind: "play", user_id: "u-1", play_label: "Implement with AI" }, "Alice")).toBe("Alice via Implement with AI");
    expect(actorLabel({ kind: "play:mcp", user_id: "u-1", play_label: "Implement with AI" }, "Alice")).toBe("Alice via Implement with AI");
    expect(actorLabel({ kind: "automation", automation_name: "Nightly sweep" }, "")).toBe("via Nightly sweep");
  });
});

describe("deltaSign", () => {
  it("reads a label or blocker as added or removed, and every other field as a move", () => {
    expect(deltaSign("labels", "", "security")).toBe("+");
    expect(deltaSign("blocked_by", "t-2", "")).toBe("−");
    expect(deltaSign("status", "", "Todo")).toBeUndefined();
  });
});

describe("dayLabel", () => {
  it("reads a day on the calendar it names, keeping the year only when it is not this one", () => {
    expect(dayLabel("2020-11-17")).toMatch(/17.*2020|2020.*17/);
    const thisYear = new Date().getFullYear();
    expect(dayLabel(`${thisYear}-11-17`)).not.toContain(String(thisYear));
    expect(dayLabel("someday")).toBe("someday");
  });
});
