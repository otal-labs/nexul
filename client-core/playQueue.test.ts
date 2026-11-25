import { describe, expect, it } from "vitest";

import { waitingLines, waitingWhy } from "@nexul/client-core/playQueue";

const queueItem = (over: { reason: string }) => ({ status: "queued", target_type: "ticket", ...over });

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
});
