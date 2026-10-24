// @vitest-environment node
import { describe, expect, it } from "vitest";

import {
  asksScope,
  canChangeMeeting,
  dayKey,
  groupReplies,
  type Meeting,
  myReply,
  occurrencesByDay,
  occurrenceTimes,
  shouldReportTimezone,
  stepAnchor,
  viewDays,
  viewRange,
} from "@nexul/client-core/calendar";

const meeting = (overrides: Partial<Meeting> = {}): Meeting => ({
  id: "m-1",
  workspace_id: "ws-1",
  organiser_id: "alice",
  title: "Standup",
  notes: "",
  starts_at: "2026-11-02T10:00:00Z",
  ends_at: "2026-11-02T10:30:00Z",
  timezone: "UTC",
  voice_channel_id: "",
  link: "",
  repeats: "none",
  remind_minutes: 10,
  invitees: [
    { user_id: "bob", reply: "accepted" },
    { user_id: "carol", reply: "declined" },
    { user_id: "dave", reply: "pending" },
  ],
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
  ...overrides,
});

describe("the calendar's days", () => {
  it("shows a month as whole weeks from Monday", () => {
    const days = viewDays("month", new Date(2026, 10, 15));
    expect(dayKey(days[0] as Date)).toBe("2026-10-26");
    expect(dayKey(days[days.length - 1] as Date)).toBe("2026-12-06");
    expect(days.length % 7).toBe(0);
  });

  it("shows a week as its seven days from Monday", () => {
    const days = viewDays("week", new Date(2026, 10, 5));
    expect(days.map(dayKey)).toEqual(["2026-11-02", "2026-11-03", "2026-11-04", "2026-11-05", "2026-11-06", "2026-11-07", "2026-11-08"]);
  });

  it("asks for the shown days up to the midnight after the last", () => {
    const { from, to } = viewRange("week", new Date(2026, 10, 5));
    expect(dayKey(from)).toBe("2026-11-02");
    expect(dayKey(to)).toBe("2026-11-09");
  });

  it("steps a week by seven days and a month to the next month's first", () => {
    expect(dayKey(stepAnchor("week", new Date(2026, 10, 5), 1))).toBe("2026-11-12");
    expect(dayKey(stepAnchor("month", new Date(2026, 0, 31), 1))).toBe("2026-02-01");
    expect(dayKey(stepAnchor("month", new Date(2026, 0, 31), -1))).toBe("2025-12-01");
  });

  it("groups occurrences by the local day they start on, in order", () => {
    const local = (d: number, h: number) => new Date(2026, 10, d, h).toISOString();
    const byDay = occurrencesByDay([
      { meeting_id: "b", slot: local(3, 15), starts_at: local(3, 15), ends_at: local(3, 16) },
      { meeting_id: "a", slot: local(3, 9), starts_at: local(3, 9), ends_at: local(3, 10) },
      { meeting_id: "c", slot: local(4, 9), starts_at: local(4, 9), ends_at: local(4, 10) },
    ]);
    expect([...byDay.keys()]).toEqual(["2026-11-03", "2026-11-04"]);
    expect(byDay.get("2026-11-03")?.map((o) => o.meeting_id)).toEqual(["a", "b"]);
  });
});

describe("a meeting's people", () => {
  it("groups the invitees by their reply", () => {
    const r = groupReplies(meeting());
    expect([r.accepted, r.declined, r.pending].map((g) => g.map((i) => i.user_id))).toEqual([["bob"], ["carol"], ["dave"]]);
    expect(myReply(meeting(), "carol")).toBe("declined");
    expect(myReply(meeting(), "alice")).toBeUndefined();
  });

  it.each([
    ["the organiser with calendar:write", "alice", ["calendar:write"], true],
    ["the organiser without it", "alice", [], false],
    ["anyone with calendar:delete", "bob", ["calendar:delete"], true],
    ["an invitee with calendar:write only", "bob", ["calendar:write"], false],
  ])("%s", (_, user, held, can) => {
    expect(canChangeMeeting(meeting(), user, (a) => held.includes(a))).toBe(can);
  });

  it("asks which occurrences only of a repeating meeting", () => {
    expect(asksScope(meeting())).toBe(false);
    expect(asksScope(meeting({ repeats: "weekly" }))).toBe(true);
  });
});

describe("the timezone report", () => {
  it.each([
    ["nothing stored yet", undefined, "Europe/London", true],
    ["the same zone", "Europe/London", "Europe/London", false],
    ["a new zone", "Europe/London", "Asia/Tokyo", true],
    ["a device that knows none", "Europe/London", undefined, false],
  ])("%s", (_, stored, device, report) => {
    expect(shouldReportTimezone(stored, device)).toBe(report);
  });
});

describe("the times of an occurrence", () => {
  it("run from the slot for as long as the meeting", () => {
    expect(occurrenceTimes(meeting({ repeats: "weekly" }), "2026-11-09T10:00:00Z")).toEqual({
      starts_at: "2026-11-09T10:00:00.000Z",
      ends_at: "2026-11-09T10:30:00.000Z",
    });
  });

  it.each([["no slot", undefined], ["a slot that isn't a time", "later"]])("fall back to the meeting's own for %s", (_, slot) => {
    expect(occurrenceTimes(meeting(), slot)).toEqual({ starts_at: "2026-11-02T10:00:00Z", ends_at: "2026-11-02T10:30:00Z" });
  });
});
