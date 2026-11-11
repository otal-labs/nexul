import { describe, expect, it } from "vitest";

import { bodyToBlocks, diffBlocks, groupRuns } from "@/utils/HistoryBlocksUtility";
import { codeRuns, foldContext, type CodeLine } from "@/utils/HistoryCodeUtility";

const diff = (before: string, after: string) => codeRuns(groupRuns(diffBlocks(bodyToBlocks(before, "code"), bodyToBlocks(after, "code")), 1, true));
const brief = (line: CodeLine) => [line.sign, line.oldNo, line.newNo, line.parts.map((p) => `${p.type}:${p.text}`).join("|")];

describe("codeRuns", () => {
  it("reads one hunk as its old lines then its new ones, numbering each side and marking an edited line's own words", () => {
    const runs = diff("a := 1\nb := 2\nreturn a\n", "a := 1\nb := 3\nlog(b)\nreturn a\n");

    expect(runs.map((run) => [run.mark, run.change])).toEqual([
      ["same", 0],
      ["edited", 1],
      ["same", 0],
    ]);
    expect(runs[1]!.lines.map(brief)).toEqual([
      ["remove", 2, undefined, "context:b := |remove:2"],
      ["add", undefined, 2, "context:b := |add:3"],
      ["add", undefined, 3, "context:log(b)"],
    ]);
    expect(runs[2]!.lines.map(brief)).toEqual([["context", 3, 4, "context:return a"]]);
  });
});

describe("foldContext", () => {
  const lines = Array.from({ length: 12 }, (_, i): CodeLine => ({ sign: "context", oldNo: i + 1, newNo: i + 1, parts: [] }));
  const sizes = (fold: ReturnType<typeof foldContext>) => [fold.head.length, fold.hidden.length, fold.tail.length];

  it.each([
    ["between two changes, three lines by each", false, false, true, [3, 6, 3]],
    ["before the first change, only the lines by it", true, false, true, [0, 9, 3]],
    ["after the last change, only the lines by it", false, true, true, [3, 9, 0]],
    ["in a version with no change, nothing", true, true, false, [12, 0, 0]],
  ] as const)("folds an unchanged stretch %s", (_, first, last, hasChanges, expected) => {
    expect(sizes(foldContext(lines, first, last, hasChanges))).toEqual(expected);
  });

  it("keeps a stretch too short to be worth a fold row", () => {
    expect(sizes(foldContext(lines.slice(0, 9), false, false, true))).toEqual([9, 0, 0]);
  });
});
