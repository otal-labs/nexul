// @vitest-environment node
import { describe, expect, it } from "vitest";

import { bodyToBlocks, diffBlocks } from "@/utils/HistoryBlocksUtility";

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
