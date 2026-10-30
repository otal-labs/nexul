import { describe, expect, it } from "vitest";

import { countLabel } from "@/utils/CountUtility";

describe("countLabel", () => {
  it.each([
    [0, "No tickets"],
    [1, "1 ticket"],
    [8, "8 tickets"],
  ])("says %i as %s", (n, want) => expect(countLabel(n, "ticket")).toBe(want));
});
