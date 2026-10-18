import { describe, expect, it } from "vitest";

import type { MentionChipData } from "@nexul/client-core/mention";

import { getChips, publishChips, subscribeChips } from "@/components/doc/mention/mentionChipsStore";

const chip: MentionChipData = { type: "ticket", id: "t-1", title: "Fix the bug", can_open: true };

describe("mentionChipsStore", () => {
  it("publishes the resolved chip map to subscribers until they unsubscribe", () => {
    publishChips(new Map());
    let notified = 0;
    const unsubscribe = subscribeChips(() => {
      notified++;
    });
    const map = new Map([[`${chip.type}:${chip.id}`, chip]]);
    publishChips(map);
    expect(notified).toBe(1);
    expect(getChips()).toBe(map);

    unsubscribe();
    publishChips(new Map());
    expect(notified).toBe(1);
  });
});
