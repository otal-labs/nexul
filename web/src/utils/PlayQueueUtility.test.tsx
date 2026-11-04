// @vitest-environment node
import { describe, expect, it } from "vitest";

import { queueItem } from "@/test/playQueueItem";
import { outcomeText, waitingLines, waitingLink, waitingWhy } from "@/utils/PlayQueueUtility";

describe("outcomeText", () => {
  it.each([
    ["nobody to run it on: the ticket has no developer", "nobody is the ticket's developer"],
    ["nobody to run it on: no person caused it and the ticket has no developer", "nobody caused it and the ticket has no developer"],
    ["nobody to run it on: the doc has no causer", "nobody caused it"],
    ['invalid: play "Fix with AI" is disabled', "the play is switched off"],
    ['invalid: play "Fix with AI" is excluded from this project', "the play is excluded from this project"],
    ['forbidden: plays:run required on play "Fix with AI"', "its person may not run the play"],
    ["get project p-1: not found", "its person can't open the project"],
    ["get play pl-1: not found", "its person can't see the play"],
    ["get ticket t-1: not found", "the ticket is gone"],
    ["get stack st-1: not found", "the stack is gone"],
    ["nobody to run it on: no one has picked a computer for it", "no one has picked a computer for it"],
    ["alice may not run agents on the computer picked for it", "alice may not run agents on the computer picked for it"],
    ["no longer unblocked", "no longer unblocked"],
    ["alice may not run Fix with AI here", "alice may not run Fix with AI here"],
    ["invalid: set up a provider on the computer first", "set up a provider on the computer first"],
  ])("says %j as %j", (reason, want) => {
    expect(outcomeText(reason)).toBe(want);
  });
});

const wsPath = (path: string) => `/acme${path}`;

describe("waitingWhy and waitingLines", () => {
  it.each([
    ["", "next in line", "next in line"],
    ["offline", "computer offline", "their computer is offline"],
    ["unpaired", "no computer paired", "no computer is paired to run it on"],
    ["expired_token", "computer session expired", "the paired computer's session expired"],
    ["no_computer_ready", "no computer can take it", "none of their computers can take it"],
    ["setup_required", "computer setup not done", "the computer's setup isn't done"],
  ])("says the held reason %j in plain words", (reason, why, sentence) => {
    const item = queueItem({ reason });
    expect([waitingWhy(item), waitingLines(item)]).toEqual([why, [sentence]]);
  });

  it("names each computer the run waits on, to its person", () => {
    const item = queueItem({
      reason: "no_computer_ready",
      waiting_on: [
        { computer_id: "c-1", name: "laptop", why: "offline", text: "laptop is offline" },
        { computer_id: "c-2", name: "build-box", why: "no_checkout", text: "build-box doesn't have this project" },
      ],
    });
    expect(waitingLines(item)).toEqual(["laptop is offline", "build-box doesn't have this project"]);
    expect(waitingLink(item, wsPath)?.to).toBe("/settings/pairing/projects");
  });

  it("reads a run waiting only for room as busy, with nothing to fix", () => {
    const full = { computer_id: "c-1", name: "laptop", why: "full" as const, text: "laptop is full (2 of 2 running)" };
    const item = queueItem({ reason: "no_computer_ready", waiting_on: [full] });
    expect([waitingWhy(item), waitingLink(item, wsPath)]).toEqual(["computers busy", undefined]);
  });

  it("sends a run held on its auto play's own computers to that auto play, not to the person's computers", () => {
    const picked = { run_on: "computer" as const, computer_ids: ["c-1"], computer_id: "c-1" };
    expect(waitingLink(queueItem({ ...picked, reason: "offline" }), wsPath)).toEqual({ to: "/acme/plays/play-1", label: "Auto play's computers" });
    expect(waitingLink(queueItem({ ...picked, reason: "no_computer_ready" }), wsPath)?.to).toBe("/acme/plays/play-1");
    expect(waitingLink(queueItem({ ...picked, reason: "setup_required" }), wsPath)?.label).toBe("Run setup");
    expect(waitingLink(queueItem({ reason: "offline" }), wsPath)).toEqual({ to: "/settings/pairing", label: "Your computers" });
  });

  it("links nothing for a hold settings can't fix", () => {
    expect(waitingLink(queueItem({ reason: "ticket busy" }), wsPath)).toBeUndefined();
    expect(waitingLink(queueItem({ reason: "offline", status: "dispatching" }), wsPath)).toBeUndefined();
  });
});
