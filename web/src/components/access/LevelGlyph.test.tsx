import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LevelGlyph } from "@/components/access/LevelGlyph";

const slots = (level: number, rungs: number) => {
  const { container } = render(<LevelGlyph level={level} rungs={rungs} />);
  return [...container.firstElementChild!.children].map((slot) => {
    if (slot.className.includes("bg-foreground")) return "full";
    return slot.className.includes("border") ? "outline" : "tick";
  });
};

describe("LevelGlyph", () => {
  it("draws a level the same way whatever the area's ladder, the rungs it could still hold outlined and the ones it lacks a tick", () => {
    expect(slots(2, 3)).toEqual(["full", "full", "outline"]);
    expect(slots(2, 2)).toEqual(["full", "full", "tick"]);
    expect(slots(1, 1)).toEqual(["full", "tick", "tick"]);
  });

  it("names each slot in its tooltip", () => {
    const { container } = render(<LevelGlyph level={2} rungs={2} />);
    expect(container.firstElementChild).toHaveAttribute("title", "Read and Write held, Delete not offered here");
  });
});
