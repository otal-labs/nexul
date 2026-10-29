// @vitest-environment node
import { describe, expect, it } from "vitest";

import { diffLines, diffWords, type DiffSegment, type DiffStats } from "@/utils/DiffUtility";

describe("diffLines", () => {
  it("returns every line as context when both sides are identical", () => {
    expect(diffLines("a\nb\nc", "a\nb\nc")).toEqual([
      { type: "context", text: "a" },
      { type: "context", text: "b" },
      { type: "context", text: "c" },
    ]);
  });

  it("marks a changed line as one remove and one add", () => {
    expect(diffLines("a\nb\nc", "a\nx\nc")).toEqual([
      { type: "context", text: "a" },
      { type: "remove", text: "b" },
      { type: "add", text: "x" },
      { type: "context", text: "c" },
    ]);
  });

  it("marks every line as added when the before side is empty", () => {
    expect(diffLines("", "a\nb")).toEqual([
      { type: "remove", text: "" },
      { type: "add", text: "a" },
      { type: "add", text: "b" },
    ]);
  });

  it("marks every line as removed when the after side is empty", () => {
    expect(diffLines("a\nb", "")).toEqual([
      { type: "remove", text: "a" },
      { type: "remove", text: "b" },
      { type: "add", text: "" },
    ]);
  });

  it("handles a pure insertion in the middle", () => {
    expect(diffLines("a\nc", "a\nb\nc")).toEqual([
      { type: "context", text: "a" },
      { type: "add", text: "b" },
      { type: "context", text: "c" },
    ]);
  });

  it("handles a pure removal in the middle", () => {
    expect(diffLines("a\nb\nc", "a\nc")).toEqual([
      { type: "context", text: "a" },
      { type: "remove", text: "b" },
      { type: "context", text: "c" },
    ]);
  });

  it("does not count a repeated line twice when the trimmed runs meet", () => {
    expect(diffLines("a\na", "a\na\na")).toEqual([
      { type: "context", text: "a" },
      { type: "context", text: "a" },
      { type: "add", text: "a" },
    ]);
  });
});

describe("diffWords", () => {
  const rejoin = (segments: DiffSegment[], keep: DiffSegment["type"]) =>
    segments
      .filter((segment) => segment.type === "context" || segment.type === keep)
      .map((segment) => segment.text)
      .join("");

  it("returns one context segment when both sides are identical", () => {
    expect(diffWords("the quick fox", "the quick fox")).toEqual([{ type: "context", text: "the quick fox" }]);
  });

  it("marks a replaced word inside a line as one remove and one add", () => {
    expect(diffWords("the quick fox", "the slow fox")).toEqual([
      { type: "context", text: "the " },
      { type: "remove", text: "quick" },
      { type: "add", text: "slow" },
      { type: "context", text: " fox" },
    ]);
  });

  it("marks an inserted word as one add", () => {
    expect(diffWords("the fox", "the quick fox")).toEqual([
      { type: "context", text: "the " },
      { type: "add", text: "quick " },
      { type: "context", text: "fox" },
    ]);
  });

  it("marks a removed word as one remove", () => {
    expect(diffWords("the quick fox", "the fox")).toEqual([
      { type: "context", text: "the " },
      { type: "remove", text: "quick " },
      { type: "context", text: "fox" },
    ]);
  });

  it("marks everything as added when the before side is empty", () => {
    expect(diffWords("", "new words")).toEqual([{ type: "add", text: "new words" }]);
  });

  it("marks everything as removed when the after side is empty", () => {
    expect(diffWords("old words", "")).toEqual([{ type: "remove", text: "old words" }]);
  });

  it("returns nothing when both sides are empty", () => {
    expect(diffWords("", "")).toEqual([]);
  });

  it.each([
    ["  leading and\ttrailing \n", "leading   and\n\nchanged trailing  "],
    ["one two  three", "one  2 three four"],
    ["", " "],
  ])("re-joins each side back to the exact original string for %j and %j", (before, after) => {
    const segments = diffWords(before, after);

    expect(rejoin(segments, "remove")).toBe(before);
    expect(rejoin(segments, "add")).toBe(after);
  });

  it("merges adjacent words of the same type into one run", () => {
    const segments = diffWords("keep one two three keep", "keep four five keep");

    expect(segments.filter((segment) => segment.type === "remove")).toHaveLength(1);
    expect(segments.filter((segment) => segment.type === "add")).toHaveLength(1);
  });

  it("builds a table sized by the edit, not by the text, for a one-word edit in 5,000 paragraphs", () => {
    const paragraphs = Array.from({ length: 5000 }, (_, i) => `paragraph ${i}`);
    const edited = [...paragraphs.slice(0, 2500), "paragraph changed", ...paragraphs.slice(2501)];
    const stats: DiffStats = { cells: 0 };

    const out = diffWords(paragraphs.join("\n"), edited.join("\n"), stats);

    expect(stats.cells).toBeLessThan(100);
    expect(out.filter((segment) => segment.type !== "context")).toEqual([
      { type: "remove", text: "2500" },
      { type: "add", text: "changed" },
    ]);
  });
});
