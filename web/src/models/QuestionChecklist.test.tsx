import { describe, expect, it } from "vitest";

import { answerLine } from "@/models/QuestionChecklist";

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
