// @vitest-environment node
import { describe, expect, it } from "vitest";

import { waitingLines, waitingWhy } from "@nexul/client-core/playQueue";

import { queueItem } from "@/test/playQueueItem";
import { outcomeText, waitingLink } from "@/utils/PlayQueueUtility";

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


describe("waitingWhy and waitingLines", () => {
  it("names each computer the run waits on, to its person", () => {
    const item = queueItem({
      reason: "no_computer_ready",
      waiting_on: [
        { computer_id: "c-1", name: "laptop", why: "offline", text: "laptop is offline" },
        { computer_id: "c-2", name: "build-box", why: "no_checkout", text: "build-box doesn't have this project" },
      ],
    });
    expect(waitingLines(item)).toEqual(["laptop is offline", "build-box doesn't have this project"]);
    expect(waitingLink(item)?.to).toBe("/agent");
  });

  it("reads a run waiting only for room as busy, with nothing to fix", () => {
    const full = { computer_id: "c-1", name: "laptop", why: "full" as const, text: "laptop is full (2 of 2 running)" };
    const item = queueItem({ reason: "no_computer_ready", waiting_on: [full] });
    expect([waitingWhy(item), waitingLink(item)]).toEqual(["computers busy", undefined]);
  });

  it("sends a run held on its person's computers to the Agent page like any other", () => {
    expect(waitingLink(queueItem({ run_on: "person", reason: "offline" }))).toEqual({ to: "/agent", label: "Your computers" });
    expect(waitingLink(queueItem({ run_on: "person", reason: "setup_required" }))?.label).toBe("Run setup");
  });

  it("links nothing for a hold settings can't fix", () => {
    expect(waitingLink(queueItem({ reason: "ticket busy" }))).toBeUndefined();
    expect(waitingLink(queueItem({ reason: "offline", status: "dispatching" }))).toBeUndefined();
  });
});
