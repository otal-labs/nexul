import { describe, expect, it } from "vitest";

import { outcomeText } from "@/utils/PlayQueueUtility";

describe("outcomeText", () => {
  it.each([
    ["nobody to run it on: the ticket has no developer", "nobody is the ticket's developer"],
    ["nobody to run it on: no person caused it and the ticket has no developer", "nobody caused it and the ticket has no developer"],
    ["nobody to run it on: the doc has no causer", "nobody caused it"],
    ['invalid: play "Fix with AI" is disabled', "the play is switched off"],
    ['invalid: play "Fix with AI" is excluded from this project', "the play is excluded from this project"],
    ['forbidden: plays:run required on play "Fix with AI"', "its person may not run the play"],
    ["get project p-1: not found", "its person can't open the project"],
    ["get ticket t-1: not found", "the ticket is gone"],
    ["no longer unblocked", "no longer unblocked"],
    ["alice may not run Fix with AI here", "alice may not run Fix with AI here"],
    ["invalid: set up a provider on the computer first", "set up a provider on the computer first"],
  ])("says %j as %j", (reason, want) => {
    expect(outcomeText(reason)).toBe(want);
  });
});
