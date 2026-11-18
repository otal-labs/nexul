import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LevelGlyph } from "@/components/access/LevelGlyph";

const slots = (level: number, rungs: number) => {
  const { container } = render(<LevelGlyph level={level} rungs={rungs} />);
  return [...container.firstElementChild!.children].map((slot) => {
    if (slot.className.includes("border")) return "outline";
    return slot.className.includes("bg-foreground") ? "full" : "dim";
  });
};

describe("LevelGlyph", () => {
  it("draws a level the same way whatever the area's ladder, with the rungs it lacks outlined after it", () => {
    expect(slots(2, 3)).toEqual(["full", "full", "dim"]);
    expect(slots(2, 2)).toEqual(["full", "full", "outline"]);
    expect(slots(1, 1)).toEqual(["full", "outline", "outline"]);
  });
});
