import { describe, expect, it } from "vitest";

import { getChips, publishChips, subscribeChips } from "@/components/doc/mention/mentionChipsStore";
import type { MentionChipData } from "@/models/Mention";

const chip: MentionChipData = { type: "ticket", id: "t-1", title: "Fix the bug", can_open: true };

describe("mentionChipsStore", () => {
  it("publishes and reads the resolved chip map", () => {
    const map = new Map([[`${chip.type}:${chip.id}`, chip]]);
    publishChips(map);
    expect(getChips()).toBe(map);
  });

  it("notifies subscribers on publish and unsubscribes", () => {
    publishChips(new Map());
    let notified = 0;
    const unsubscribe = subscribeChips(() => {
      notified++;
    });
    publishChips(new Map([[`${chip.type}:${chip.id}`, chip]]));
    expect(notified).toBe(1);

    unsubscribe();
    publishChips(new Map());
    expect(notified).toBe(1);
  });
});
