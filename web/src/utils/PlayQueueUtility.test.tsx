import { describe, expect, it } from "vitest";

import { queueItem } from "@/test/playQueueItem";
import { outcomeText, pickingComputer, waitingSentence, waitingWhy } from "@/utils/PlayQueueUtility";

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

describe("waitingWhy and waitingSentence", () => {
  it.each([
    ["", "waiting its turn", "waiting its turn"],
    ["no free slot", "busy with another run", "busy with another run"],
    ["offline", "computer offline", "computer offline"],
    ["unpaired", "no computer paired", "no computer is paired to run it on; add one in Settings → Computers"],
    ["expired_token", "computer session expired", "the paired computer's session expired; pair it again in Settings → Computers"],
    ["no_default", "no T3 project linked", "no T3 project is picked for this project; link one in Settings → Computers → Projects"],
    ["no_default_computer", "no default computer", "several computers are paired and none is the default; pick one, and this project's runs use it from now on"],
    ["setup_required", "computer setup not done", "the computer's setup isn't done; run it in Settings → Computers"],
  ])("says the held reason %j in plain words", (reason, why, sentence) => {
    const item = queueItem({ reason });
    expect([waitingWhy(item), waitingSentence(item)]).toEqual([why, sentence]);
  });

  it.each([
    [{ reason: "no_default_computer" }, true],
    [{ reason: "offline", computer_id: "c-laptop" }, true],
    [{ reason: "ticket busy", computer_id: "c-laptop" }, true],
    [{ reason: "no_default_computer", status: "dispatching" as const }, false],
    [{ reason: "", computer_id: "c-laptop", status: "dispatching" as const }, false],
    [{ reason: "unpaired" }, false],
    [{ reason: "offline" }, false],
  ])("asks for a computer while a run waits for no default computer or on its pick (%j)", (overrides, want) => {
    expect(pickingComputer(queueItem(overrides))).toBe(want);
  });
});
