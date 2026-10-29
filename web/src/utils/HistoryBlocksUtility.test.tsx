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
