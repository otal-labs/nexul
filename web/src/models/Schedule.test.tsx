// @vitest-environment node
import { describe, expect, it } from "vitest";

import { newDraft, toSaveRequest } from "@/models/AutoPlay";
import { countsPhrase } from "@/models/Schedule";

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
