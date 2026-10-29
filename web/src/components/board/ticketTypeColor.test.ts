import { describe, expect, it } from "vitest";

import { labelDotColor, ticketTypeColor } from "@/components/board/ticketTypeColor";

describe("ticketTypeColor", () => {
  it("returns a known text color for a recognized type, no background/border fill", () => {
    expect(ticketTypeColor("bug")).toMatch(/fuchsia/);
    expect(ticketTypeColor("Bug")).toEqual(ticketTypeColor("bug"));
    expect(ticketTypeColor("bug")).not.toMatch(/bg-|border-/);
  });

  it("can return different colors for different unrecognized types", () => {
    const colors = new Set(["alpha", "beta", "gamma", "delta", "epsilon"].map((t) => ticketTypeColor(t)));
    expect(colors.size).toBeGreaterThan(1);
  });

  it("prefers a configured backend color over the hash, even for a recognized type", () => {
    expect(ticketTypeColor("bug", "cyan")).toMatch(/cyan/);
    expect(ticketTypeColor("bug", "cyan")).not.toBe(ticketTypeColor("bug"));
  });

  it("falls back to the hash when no color is configured (undefined or empty)", () => {
    expect(ticketTypeColor("bug", undefined)).toBe(ticketTypeColor("bug"));
    expect(ticketTypeColor("bug", "")).toBe(ticketTypeColor("bug"));
  });

  it("falls back to the hash for an unrecognized configured color instead of crashing", () => {
    expect(ticketTypeColor("bug", "magenta")).toBe(ticketTypeColor("bug"));
  });
});

describe("labelDotColor", () => {
  it("returns a solid bg color (dot fill), case-insensitively stable", () => {
    expect(labelDotColor("urgent")).toMatch(/^bg-/);
    expect(labelDotColor("Urgent")).toEqual(labelDotColor("urgent"));
  });

  it("can return different colors for different labels", () => {
    const colors = new Set(["alpha", "beta", "gamma", "delta", "epsilon"].map((l) => labelDotColor(l)));
    expect(colors.size).toBeGreaterThan(1);
  });

  it("prefers a configured backend color over the hash", () => {
    expect(labelDotColor("urgent", "orange")).toBe("bg-orange-500");
    expect(labelDotColor("urgent", "orange")).not.toBe(labelDotColor("urgent"));
  });

  it("falls back to the hash when no color is configured (undefined or empty)", () => {
    expect(labelDotColor("urgent", undefined)).toBe(labelDotColor("urgent"));
    expect(labelDotColor("urgent", "")).toBe(labelDotColor("urgent"));
  });
});
