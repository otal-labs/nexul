import type { AnswerValue, QuestionItem } from "@/models/Question";

// What a checklist row needs of a stored answer: no picks, no text, and not skipped is pending.
export interface ChecklistAnswer {
  selected: string[];
  text: string;
  skipped: boolean;
}

export type ChecklistRowStatus = "answered" | "skipped" | "pending";

// open: a draft waiting on an unanswered or skipped question; confirmed: the answer is the draft; suggested: newer and different.
export type ChecklistDraftState = "open" | "confirmed" | "suggested";

// What a row shows of a drafted answer: its state and where it came from.
export interface ChecklistDraft {
  state: ChecklistDraftState;
  from: string;
}

// One question of a checklist (the Interview page, a doc's Questions), whatever stores its answer.
export interface ChecklistRow {
  key: string;
  item: QuestionItem;
  why: string;
  answer: ChecklistAnswer | undefined;
  draft?: ChecklistDraft | undefined;
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
  const drafted = rows.filter((r) => r.draft?.state === "open").length;
  const suggested = rows.filter((r) => r.draft?.state === "suggested").length;
  const head = skipped === 0 ? `${answered} of ${rows.length} answered` : `${answered} answered · ${skipped} skipped`;
  return [head, drafted > 0 && `${drafted} drafted`, suggested > 0 && `${suggested} suggested`].filter(Boolean).join(" · ");
};

// A row waits on the person while it is unanswered or carries a suggested change to accept or dismiss.
export const firstPendingKey = (rows: ChecklistRow[]): string | null =>
  rows.find((r) => rowStatus(r) === "pending" || r.draft?.state === "suggested")?.key ?? null;

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

export const valueLine = (value: AnswerValue | undefined): string =>
  value ? [...(value.selected ?? []), value.text ?? ""].filter((part) => part !== "").join(" · ").replaceAll("\n", " · ") : "";

// A doc question's suggested option keeps its "(Suggested)" mark as picked; the one-line answer reads without it.
const withoutSuggested = (label: string): string => label.replace(/\s*\(Suggested\)$/, "");

export const answerLine = (answer: ChecklistAnswer | undefined): string =>
  answer ? valueLine({ selected: answer.selected.map(withoutSuggested), text: answer.text }) : "";
