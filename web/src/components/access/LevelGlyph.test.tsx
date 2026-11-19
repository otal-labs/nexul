import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LevelGlyph } from "@/components/access/LevelGlyph";

const slots = (level: number, rungs: number) => {
  const { container } = render(<LevelGlyph level={level} rungs={rungs} />);
  return [...container.firstElementChild!.children].map((slot) => {
    if (slot.className.includes("bg-foreground")) return "full";
    return slot.className.includes("border") ? "outline" : "empty";
  });
};

describe("LevelGlyph", () => {
  it("draws a level the same way whatever the area's ladder, the rungs it could still hold outlined and the ones it lacks left empty", () => {
    expect(slots(2, 3)).toEqual(["full", "full", "outline"]);
    expect(slots(2, 2)).toEqual(["full", "full", "empty"]);
    expect(slots(1, 1)).toEqual(["full", "empty", "empty"]);
  });
});
