import { describe, expect, it } from "vitest";

import { labelDotColor, pillClass, ticketTypeColor } from "@/components/board/ticketTypeColor";

describe("ticketTypeColor", () => {
  it("returns a known text color for a recognized type, no background/border fill", () => {
    expect(ticketTypeColor("bug")).toMatch(/red/);
    expect(ticketTypeColor("Bug")).toEqual(ticketTypeColor("bug"));
    expect(ticketTypeColor("bug")).not.toMatch(/bg-|border-/);
  });

  it("returns a stable color for an unrecognized custom type", () => {
    const first = ticketTypeColor("widget");
    const second = ticketTypeColor("widget");
    expect(first).toEqual(second);
  });

  it("can return different colors for different unrecognized types", () => {
    const colors = new Set(["alpha", "beta", "gamma", "delta", "epsilon"].map((t) => ticketTypeColor(t)));
    expect(colors.size).toBeGreaterThan(1);
  });

  it("prefers a configured backend color over the hash, even for a recognized type", () => {
    expect(ticketTypeColor("bug", "cyan")).toBe("text-cyan-400");
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

  it("returns a stable color for the same label across calls", () => {
    expect(labelDotColor("needs-review")).toEqual(labelDotColor("needs-review"));
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

describe("pillClass", () => {
  it("tints a pill from the hue of a text or bg color class", () => {
    expect(pillClass("text-cyan-400")).toBe(pillClass("bg-cyan-500"));
    expect(pillClass("text-cyan-400")).toMatch(/bg-cyan-500\/15/);
    expect(pillClass("text-red-400")).not.toBe(pillClass("text-cyan-400"));
  });

  it("falls back to the neutral slate pill for anything it can't parse", () => {
    expect(pillClass("nonsense")).toBe(pillClass("text-slate-400"));
  });
});
