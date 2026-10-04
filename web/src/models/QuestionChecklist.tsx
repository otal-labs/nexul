import type { AnswerValue, QuestionItem } from "@/models/Question";

// What a checklist row needs of a stored answer: no picks, no text, and not skipped is pending.
export interface ChecklistAnswer {
  selected: string[];
  text: string;
  skipped: boolean;
}

export type ChecklistRowStatus = "answered" | "skipped" | "pending";

// One question of a checklist (the Interview page, a doc's Questions), whatever stores its answer.
export interface ChecklistRow {
  key: string;
  item: QuestionItem;
  why: string;
  answer: ChecklistAnswer | undefined;
}

export const rowStatus = (row: ChecklistRow): ChecklistRowStatus => {
  if (!row.answer) return "pending";
  if (row.answer.skipped) return "skipped";
  if (row.answer.selected.length > 0 || row.answer.text !== "") return "answered";
  return "pending";
};

export const countLine = (rows: ChecklistRow[]): string => {
  const answered = rows.filter((r) => rowStatus(r) === "answered").length;
  const skipped = rows.filter((r) => rowStatus(r) === "skipped").length;
  if (skipped === 0) return `${answered} of ${rows.length} answered`;
  return `${answered} answered · ${skipped} skipped`;
};

export const firstPendingKey = (rows: ChecklistRow[]): string | null =>
  rows.find((r) => rowStatus(r) === "pending")?.key ?? null;

// The next row still pending once the row at key is done, wrapping round to the first one left.
export const nextPendingKey = (rows: ChecklistRow[], key: string): string | null => {
  const at = rows.findIndex((r) => r.key === key);
  const ordered = [...rows.slice(at + 1), ...rows.slice(0, Math.max(at, 0))];
  return firstPendingKey(ordered);
};

export const answerValue = (answer: ChecklistAnswer | undefined): AnswerValue | undefined => {
  if (!answer || answer.skipped) return undefined;
  return { selected: answer.selected, text: answer.text };
};

// A doc question's suggested option keeps its "(Suggested)" mark as picked; the one-line answer reads without it.
const withoutSuggested = (label: string): string => label.replace(/\s*\(Suggested\)$/, "");

export const answerLine = (answer: ChecklistAnswer | undefined): string =>
  answer
    ? [...answer.selected.map(withoutSuggested), answer.text].filter((part) => part !== "").join(" · ").replaceAll("\n", " · ")
    : "";
