// @vitest-environment node
import { describe, expect, it } from "vitest";

import { answerLine, followOpenKey, openRow, type ChecklistRow } from "@nexul/client-core/questionChecklist";

describe("answerLine", () => {
  it("reads a picked suggested option without its (Suggested) mark, keeping the rest of the answer", () => {
    const answer = { selected: ["Text message (Suggested)", "Email"], text: "Also a call", skipped: false };
    expect(answerLine(answer)).toBe("Text message · Email · Also a call");
  });

  it("leaves a (Suggested) that is not the label's ending, and the interview's (Recommended), as written", () => {
    const answer = { selected: ["(Suggested) times only", "Weekly (Recommended)"], text: "", skipped: false };
    expect(answerLine(answer)).toBe("(Suggested) times only · Weekly (Recommended)");
  });
});

const checklistRow = (key: string, answer: { selected?: string[]; text?: string; skipped?: boolean } = {}): ChecklistRow => ({
  key, item: { id: key, text: key, options: [] }, why: "", answer: { selected: [], text: "", skipped: false, ...answer },
});

describe("followOpenKey", () => {
  const before = [checklistRow("q1", { text: "Staff" }), checklistRow("q2"), checklistRow("q3")];
  const after = [before[0]!, checklistRow("q2", { text: "Customers" }), before[2]!];

  it("moves a row opened unanswered to the next one pending once someone else answers it", () => {
    const open = openRow(before, "q2");
    expect(followOpenKey(before, open, null)).toBe("q2");
    expect(followOpenKey(after, open, null)).toBe("q3");
  });

  it("moves on when the row is skipped, and to nothing once none is left", () => {
    const open = openRow(before, "q2");
    expect(followOpenKey([before[0]!, checklistRow("q2", { skipped: true }), before[2]!], open, null)).toBe("q3");
    expect(followOpenKey([before[0]!, after[1]!, checklistRow("q3", { text: "x" })], open, null)).toBeNull();
  });

  it("leaves the row the person has typed into, and a row they opened already answered", () => {
    expect(followOpenKey(after, openRow(before, "q2"), "q2")).toBe("q2");
    expect(followOpenKey(after, openRow(after, "q2"), null)).toBe("q2");
  });
});
