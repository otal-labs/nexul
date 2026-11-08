// @vitest-environment node
import { describe, expect, it } from "vitest";

import { newDraft, toSaveRequest } from "@/models/AutoPlay";
import { countsPhrase, describeSchedule, zoneName } from "@/models/Schedule";

describe("describeSchedule", () => {
  it.each([
    ["0 23 * * *", "Europe/London", "Every day at 23:00, London"],
    ["30 22 * * 1-5", "Europe/London", "Weekdays at 22:30, London"],
    ["0 9 * * sat,sun", "UTC", "Weekends at 09:00, UTC"],
    ["0 6 * * 1,4", "America/New_York", "Mondays and Thursdays at 06:00, New York"],
    ["15 * * * *", "UTC", "Every hour at :15, UTC"],
    ["0 2 1 * *", "UTC", "On day 1 of each month at 02:00, UTC"],
    ["0 23 * 1 *", "UTC", "A custom schedule, UTC"],
    ["*/2 23 * * *", "UTC", "A custom schedule, UTC"],
    ["0 23 * *", "UTC", ""],
  ])("reads %s in %s as %j", (cron, zone, words) => {
    expect(describeSchedule(cron, zone)).toBe(words);
  });

  it("names a zone by its city", () => {
    expect(zoneName("America/Argentina/Buenos_Aires")).toBe("Buenos Aires");
  });
});

describe("a schedule auto play's save", () => {
  it("sends the schedule only for the schedule moment, and a resource only while one at a time is on", () => {
    const draft = { ...newDraft("ticket"), moment: "schedule" as const, cron: " 0 23 * * * ", timezone: "Europe/London", resource: " vm " };
    expect(toSaveRequest(draft)).toMatchObject({
      schedule: { cron: "0 23 * * *", timezone: "Europe/London", stop_at: "07:00", max_runs: 10 },
      one_at_a_time: false,
      resource: "",
    });
    expect(toSaveRequest({ ...draft, one_at_a_time: true }).resource).toBe("vm");
    expect(toSaveRequest({ ...draft, moment: "ticket.created" }).schedule).toBeNull();
  });
});

describe("countsPhrase", () => {
  it("names only what happened", () => {
    expect(countsPhrase({ finished: 4, asked: 1, failed: 0, skipped: 2, didnt_run: 0 })).toBe("4 finished, 1 asked, 2 skipped");
  });
});
