import { describe, expect, it } from "vitest";

import { describeSchedule, normalRunOn, runOnPhrase, zoneName } from "@nexul/client-core/autoPlayText";

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

describe("runOnPhrase", () => {
  it("names the person whose computers a person auto play runs on, or nobody yet", () => {
    expect(runOnPhrase("person", "Your")).toBe("Your computers");
    expect(runOnPhrase("person", "Sam's")).toBe("Sam's computers");
    expect(runOnPhrase("person", "")).toBe("Nobody yet");
    expect(runOnPhrase("developer", "")).toBe("The developer's computers");
  });

  it("reads the older computer run-on as a person", () => {
    expect(normalRunOn("computer")).toBe("person");
    expect(runOnPhrase("computer", "Sam's")).toBe("Sam's computers");
  });
});
