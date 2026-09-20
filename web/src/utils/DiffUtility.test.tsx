import { describe, expect, it } from "vitest";

import { diffLines } from "@/utils/DiffUtility";

describe("diffLines", () => {
  it("returns every line as context when both sides are identical", () => {
    const result = diffLines("a\nb\nc", "a\nb\nc");
    expect(result).toEqual([
      { type: "context", text: "a" },
      { type: "context", text: "b" },
      { type: "context", text: "c" },
    ]);
  });

  it("marks a changed line as one remove and one add", () => {
    const result = diffLines("a\nb\nc", "a\nx\nc");
    expect(result).toEqual([
      { type: "context", text: "a" },
      { type: "remove", text: "b" },
      { type: "add", text: "x" },
      { type: "context", text: "c" },
    ]);
  });

  it("marks every line as added when the before side is empty", () => {
    const result = diffLines("", "a\nb");
    expect(result).toEqual([
      { type: "remove", text: "" },
      { type: "add", text: "a" },
      { type: "add", text: "b" },
    ]);
  });

  it("marks every line as removed when the after side is empty", () => {
    const result = diffLines("a\nb", "");
    expect(result).toEqual([
      { type: "remove", text: "a" },
      { type: "remove", text: "b" },
      { type: "add", text: "" },
    ]);
  });

  it("handles a pure insertion in the middle", () => {
    const result = diffLines("a\nc", "a\nb\nc");
    expect(result).toEqual([
      { type: "context", text: "a" },
      { type: "add", text: "b" },
      { type: "context", text: "c" },
    ]);
  });
});
