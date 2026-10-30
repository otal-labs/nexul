import { describe, expect, it } from "vitest";

import { hexToHsv, hsvToHex } from "@/lib/hsv";

describe("hsv", () => {
  it.each(["#10b981", "#ef4444", "#6d5bd0", "#000000", "#ffffff", "#84cc16"])("round-trips %s", (hex) => {
    expect(hsvToHex(hexToHsv(hex))).toBe(hex);
  });

  it("keeps the given hue for a grey", () => {
    expect(hexToHsv("#808080", 200)[0]).toBe(200);
  });
});
