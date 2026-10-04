import { describe, expect, it } from "vitest";

import { buildSections, type InterviewAnswer } from "@/models/InterviewAnswer";
import { countLine, firstPendingKey, nextPendingKey } from "@/models/QuestionChecklist";
import type { InterviewQuestion } from "@/models/InterviewTemplate";

const question = (text: string): InterviewQuestion => ({ text, hint: "", multi_select: false, options: [] });

const answer = (round: number, text: string, patch: Partial<InterviewAnswer> = {}): InterviewAnswer => ({
  id: `${round}-${text}`,
  workspace_id: "ws-1",
  project_id: "p-1",
  round,
  question: text,
  selected: [],
  text: "yes",
  skipped: false,
  answered_by: "u-1",
  answered_at: "",
  ...patch,
});

describe("buildSections", () => {
  it("matches answers to the template by trimmed text and drops answers to reworded questions", () => {
    const [initial] = buildSections([question("Stack? "), question("Tests?")], [answer(0, "Stack?"), answer(0, "Old wording")]);
    expect(initial?.rows.map((r) => r.answer?.question)).toEqual(["Stack?", undefined]);
  });

  it("adds one section per follow-up round, numbered in order", () => {
    const sections = buildSections([question("Stack?")], [answer(2, "B"), answer(1, "A")]);
    expect(sections.map((s) => s.label)).toEqual(["Initial questions", "Follow-ups from the agent 1", "Follow-ups from the agent 2"]);
    expect(sections[1]?.rows[0]?.item.text).toBe("A");
  });
});

describe("countLine", () => {
  it("counts answered of all until something is skipped, then answered and skipped", () => {
    const [plain] = buildSections([question("A"), question("B")], [answer(0, "A")]);
    const [skipping] = buildSections([question("A"), question("B")], [answer(0, "A"), answer(0, "B", { skipped: true, text: "" })]);
    expect(countLine(plain?.rows ?? [])).toBe("1 of 2 answered");
    expect(countLine(skipping?.rows ?? [])).toBe("1 answered · 1 skipped");
  });

  it("treats a cleared follow-up, kept with no answer, as pending", () => {
    const sections = buildSections([], [answer(1, "Why?", { text: "" })]);
    expect(countLine(sections[1]?.rows ?? [])).toBe("0 of 1 answered");
  });
});

describe("pending rows", () => {
  const [initial] = buildSections(
    [question("A"), question("B"), question("C"), question("D")],
    [answer(0, "A"), answer(0, "B", { skipped: true, text: "" }), answer(0, "D")],
  );
  const rows = initial?.rows ?? [];

  it("starts at the first question neither answered nor skipped", () => {
    expect(firstPendingKey(rows)).toBe("0:C");
  });

  it("counts a suggested change on an answered question as waiting on the person", () => {
    const suggestion = {
      id: "d-1", workspace_id: "ws-1", project_id: "p-1", question: "B", selected: [], text: "no", source_ids: [], where: "",
      drafted_by: "u-1", drafted_at: "2026-10-04T10:00:00Z",
    };
    const [all] = buildSections([question("A"), question("B")], [answer(0, "A"), answer(0, "B", { answered_at: "2026-10-03T10:00:00Z" })], [suggestion]);
    expect(firstPendingKey(all?.rows ?? [])).toBe("0:B");
  });

  it("moves on to the next pending row, wrapping to the start, and stops when none is left", () => {
    expect(nextPendingKey(rows, "0:D")).toBe("0:C");
    expect(nextPendingKey(rows, "0:C")).toBeNull();
  });
});
