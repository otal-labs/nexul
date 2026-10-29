// @vitest-environment node
import { describe, expect, it } from "vitest";

import { bodyToBlocks, diffBlocks, groupRuns } from "@/utils/HistoryBlocksUtility";

const text = (t: string) => ({ type: "text", text: t });
const p = (t: string) => ({ type: "paragraph", content: [text(t)] });
const h = (t: string) => ({ type: "heading", attrs: { level: 2 }, content: [text(t)] });
const ol = (...items: string[]) => ({ type: "orderedList", attrs: { start: 1 }, content: items.map((t) => ({ type: "listItem", content: [p(t)] })) });
const body = (...content: unknown[]) => JSON.stringify({ type: "doc", content });

const marks = (before: string, after: string) => diffBlocks(bodyToBlocks(before), bodyToBlocks(after)).map((r) => [r.mark, r.block.text]);

describe("diffBlocks", () => {
  it("shows a reworded paragraph as one edited block with the words that changed", () => {
    const before = body(h("Release day"), p("Watch error rates for an hour before promoting."), p("Close the ticket."));
    const after = body(h("Release day"), p("Watch error rates for thirty minutes before promoting."), p("Close the ticket."));

    const rows = diffBlocks(bodyToBlocks(before), bodyToBlocks(after));

    expect(rows.map((r) => r.mark)).toEqual(["same", "edited", "same"]);
    expect(rows[1]!.parts).toEqual([
      { type: "context", text: "Watch error rates for " },
      { type: "remove", text: "an hour" },
      { type: "add", text: "thirty minutes" },
      { type: "context", text: " before promoting." },
    ]);
  });

  it("keeps an unrelated added paragraph apart from the removed one instead of drawing one garbled edit", () => {
    const before = body(p("Ask Sam to sign off on the database migration."), p("Close the ticket."));
    const after = body(p("Post the freeze notice in the releases channel."), p("Close the ticket."));

    expect(marks(before, after)).toEqual([
      ["removed", "Ask Sam to sign off on the database migration."],
      ["added", "Post the freeze notice in the releases channel."],
      ["same", "Close the ticket."],
    ]);
  });

  it("marks only the new item when one is inserted at the top of a numbered list", () => {
    const before = body(ol("Merge the tickets", "Tag the branch"));
    const after = body(ol("Post the notice", "Merge the tickets", "Tag the branch"));

    expect(marks(before, after)).toEqual([
      ["added", "Post the notice"],
      ["same", "Merge the tickets"],
      ["same", "Tag the branch"],
    ]);
  });
});

describe("groupRuns", () => {
  it("makes one change of adjacent blocks with the same mark, so the stepper counts the bars on screen", () => {
    const before = body(p("Keep this."), p("Alpha beta gamma delta."), p("Epsilon zeta eta theta."), p("Reword the old plan today."), p("Closing line."));
    const after = body(p("Keep this."), p("Reword the new plan today."), p("Closing line."), p("Fresh paragraph about nothing related."), p("Another fresh paragraph on cats."));

    const runs = groupRuns(diffBlocks(bodyToBlocks(before), bodyToBlocks(after)));

    expect(runs.map((r) => [r.mark, r.rows.length, r.change])).toEqual([
      ["same", 1, 0],
      ["removed", 2, 1],
      ["edited", 1, 2],
      ["same", 1, 0],
      ["added", 2, 3],
    ]);
  });
});

describe("bodyToBlocks on plain text", () => {
  it("cuts markdown at blank lines and reads headings, list items and fences from their markers", () => {
    const text = [
      "# Fix the ticket",
      "",
      "Read the ticket first,",
      "then the code it names.",
      "",
      "- Run the tests",
      "  before pushing",
      "  - [x] Lint",
      "2. Open the pull request",
      "",
      "```sh",
      "make test",
      "",
      "make lint",
      "```",
    ].join("\n");

    expect(bodyToBlocks(text, "markdown").map((b) => [b.kind, b.text, b.level, b.depth])).toEqual([
      ["heading", "Fix the ticket", 1, 0],
      ["paragraph", "Read the ticket first, then the code it names.", 0, 0],
      ["bullet", "Run the tests before pushing", 0, 0],
      ["done", "Lint", 0, 1],
      ["ordered", "Open the pull request", 0, 0],
      ["code", "make test\n\nmake lint", 0, 0],
    ]);
  });

  it("keeps a one-word edit inside a long paragraph as one edited run with just that word marked", () => {
    const long = "Read the ticket and the files it names, then write the smallest change that makes the failing test pass, and run the whole suite before you push anything";
    const before = `${long}.\n\nClose the ticket.`;
    const after = `${long.replace("smallest", "tightest")}.\n\nClose the ticket.`;

    const runs = groupRuns(diffBlocks(bodyToBlocks(before, "markdown"), bodyToBlocks(after, "markdown")));

    expect(runs.map((r) => [r.mark, r.rows.length])).toEqual([
      ["edited", 1],
      ["same", 1],
    ]);
    expect(runs[0]!.rows[0]!.parts.filter((part) => part.type !== "context")).toEqual([
      { type: "remove", text: "smallest" },
      { type: "add", text: "tightest" },
    ]);
  });

  it("makes every line of code its own block, so a changed line is marked alone", () => {
    const rows = diffBlocks(bodyToBlocks("a := 1\nb := 2\nreturn a\n", "code"), bodyToBlocks("a := 1\nb := 3\nreturn a\n", "code"));

    expect(rows.map((r) => [r.mark, r.block.kind, r.block.text])).toEqual([
      ["same", "line", "a := 1"],
      ["edited", "line", "b := 3"],
      ["same", "line", "return a"],
    ]);
  });
});
