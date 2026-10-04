import type { InterviewQuestion } from "@/models/InterviewTemplate";
import { draftValue, sourceName, type InterviewDraft, type InterviewSource } from "@/models/InterviewSource";
import { optionValue, type AnswerValue, type HarnessQuestion, type QuestionAnswers, type QuestionItem } from "@/models/Question";
import type { Trail } from "@/models/Trail";

// Mirrors internal/plays skippedAnswer: a skipped live follow-up is sent as this text, since a harness may refuse an empty answer.
export const SKIPPED_ANSWER = "Skipped";

// Mirrors memories.InterviewAnswer: round 0 answers the template's questions, 1 and up are the agent's follow-up rounds.
export interface InterviewAnswer {
  id: string;
  workspace_id: string;
  project_id: string;
  round: number;
  question: string;
  options?: { label: string; description?: string }[];
  multi_select?: boolean;
  why?: string;
  selected: string[];
  text: string;
  skipped: boolean;
  answered_by: string;
  answered_at: string;
}

export interface SaveInterviewAnswerInput {
  project_id: string;
  round: number;
  question: string;
  selected: string[];
  text: string;
  skip: boolean;
}

export type InterviewRowStatus = "answered" | "skipped" | "pending";

// open: a draft waiting on an unanswered or skipped question; confirmed: the answer is the draft; suggested: newer and different.
export type RowDraftState = "open" | "confirmed" | "suggested";

export interface RowDraft {
  id: string;
  value: AnswerValue;
  from: string;
  where: string;
  state: RowDraftState;
}

export interface InterviewRow {
  key: string;
  round: number;
  item: QuestionItem;
  why: string;
  answer: InterviewAnswer | undefined;
  draft?: RowDraft | undefined;
  // A follow-up of the run's live question, answered on the trail with the rest of its round rather than saved alone.
  live?: true;
}

export interface InterviewSectionData {
  key: string;
  label: string;
  rows: InterviewRow[];
}

const rowKey = (round: number, question: string): string => `${round}:${question}`;

export const rowStatus = (row: InterviewRow): InterviewRowStatus => {
  if (!row.answer) return "pending";
  if (row.answer.skipped) return "skipped";
  if (row.answer.selected.length > 0 || row.answer.text !== "") return "answered";
  return "pending";
};

const sameValue = (draft: InterviewDraft, answer: InterviewAnswer): boolean =>
  draft.text.trim() === answer.text.trim() && [...draft.selected].sort().join("\n") === [...answer.selected].sort().join("\n");

// A draft as its question's row shows it; a draft older than an answer that differs from it was overridden and is not shown.
export const rowDraft = (draft: InterviewDraft | undefined, answer: InterviewAnswer | undefined, sources: InterviewSource[]): RowDraft | undefined => {
  if (!draft) return undefined;
  const from = draft.source_ids
    .map((id) => sources.find((s) => s.id === id))
    .filter((s) => s !== undefined)
    .map(sourceName)
    .join(", ");
  const base = { id: draft.id, value: draftValue(draft), from, where: draft.where };
  const answered = !!answer && !answer.skipped && (answer.selected.length > 0 || answer.text !== "");
  if (!answered) return { ...base, state: "open" };
  if (sameValue(draft, answer)) return { ...base, state: "confirmed" };
  if (Date.parse(draft.drafted_at) > Date.parse(answer.answered_at)) return { ...base, state: "suggested" };
  return undefined;
};

// The template's questions in template order, matched to their answers and drafts by trimmed text, then one section per stored round.
export const buildSections = (
  questions: InterviewQuestion[],
  answers: InterviewAnswer[],
  drafts: InterviewDraft[] = [],
  sources: InterviewSource[] = [],
): InterviewSectionData[] => {
  const initial = answers.filter((a) => a.round === 0);
  const answerTo = (q: InterviewQuestion) => initial.find((a) => a.question === q.text.trim());
  const sections: InterviewSectionData[] = [
    {
      key: "0",
      label: "Initial questions",
      rows: questions.map((q) => ({
        key: rowKey(0, q.text.trim()),
        round: 0,
        item: {
          id: rowKey(0, q.text.trim()),
          text: q.text,
          header: q.hint,
          multi_select: q.multi_select,
          options: q.options.map((o) => ({ label: o.label, description: o.description })),
        },
        why: "",
        answer: answerTo(q),
        draft: rowDraft(
          drafts.find((d) => d.question === q.text.trim()),
          answerTo(q),
          sources,
        ),
      })),
    },
  ];
  const rounds = [...new Set(answers.filter((a) => a.round > 0).map((a) => a.round))].sort((x, y) => x - y);
  rounds.forEach((round, i) => {
    sections.push({
      key: String(round),
      label: `Follow-ups from the agent ${i + 1}`,
      rows: answers
        .filter((a) => a.round === round)
        .map((a) => ({
          key: rowKey(round, a.question),
          round,
          item: { id: rowKey(round, a.question), text: a.question, multi_select: a.multi_select ?? false, options: a.options ?? [] },
          why: a.why ?? "",
          answer: a,
        })),
    });
  });
  return sections;
};

export const countLine = (rows: InterviewRow[]): string => {
  const answered = rows.filter((r) => rowStatus(r) === "answered").length;
  const skipped = rows.filter((r) => rowStatus(r) === "skipped").length;
  const drafted = rows.filter((r) => r.draft?.state === "open").length;
  const suggested = rows.filter((r) => r.draft?.state === "suggested").length;
  const head = skipped === 0 ? `${answered} of ${rows.length} answered` : `${answered} answered · ${skipped} skipped`;
  return [head, drafted > 0 && `${drafted} drafted`, suggested > 0 && `${suggested} suggested`].filter(Boolean).join(" · ");
};

export const firstPendingKey = (rows: InterviewRow[]): string | null =>
  rows.find((r) => rowStatus(r) === "pending")?.key ?? null;

// The next row still pending once the row at key is done, wrapping round to the first one left.
export const nextPendingKey = (rows: InterviewRow[], key: string): string | null => {
  const at = rows.findIndex((r) => r.key === key);
  const ordered = [...rows.slice(at + 1), ...rows.slice(0, Math.max(at, 0))];
  return firstPendingKey(ordered);
};

export const answerValue = (answer: InterviewAnswer | undefined): AnswerValue | undefined => {
  if (!answer || answer.skipped) return undefined;
  return { selected: answer.selected, text: answer.text };
};

export const valueLine = (value: AnswerValue | undefined): string =>
  value ? [...(value.selected ?? []), value.text ?? ""].filter((part) => part !== "").join(" · ").replaceAll("\n", " · ") : "";

export const answerLine = (answer: InterviewAnswer | undefined): string => valueLine(answer);

// Mirrors internal/plays splitWhy: a follow-up's text is the question up to the first "?" followed by whitespace, then why it is asked.
export const splitWhy = (text: string): { question: string; why: string } => {
  const match = /\?\s/.exec(text);
  if (!match) return { question: text, why: "" };
  return { question: text.slice(0, match.index + 1), why: text.slice(match.index + 1).trim() };
};

// What the agent labelled (Recommended), picked before the person touches the question: the first one, or each for a multi-select.
export const recommendedDraft = (item: QuestionItem): AnswerValue | undefined => {
  const picks = item.options.filter((o) => o.label.includes("(Recommended)")).map(optionValue);
  const chosen = item.multi_select ? picks : picks.slice(0, 1);
  if (chosen.length === 0) return undefined;
  return { selected: chosen };
};

const draftAnswer = (round: number, question: string, value: AnswerValue): InterviewAnswer => {
  const selected = value.selected ?? [];
  const text = value.text ?? "";
  return {
    id: "", workspace_id: "", project_id: "", round, question, selected, text,
    skipped: selected.length === 0 && (text.trim() === "" || text.trim() === SKIPPED_ANSWER), answered_by: "", answered_at: "",
  };
};

// The question a waiting interview run asks, as the next round's rows; the header is the harness's chip and is left out.
export const liveSection = (question: HarnessQuestion, stored: InterviewSectionData[], drafts: QuestionAnswers): InterviewSectionData => {
  const round = Math.max(0, ...stored.flatMap((s) => s.rows.map((r) => r.round))) + 1;
  return {
    key: `live:${question.request_id}`,
    label: `Follow-ups from the agent ${round}`,
    rows: question.questions.map((item) => {
      const { question: text, why } = splitWhy(item.text);
      const draft = drafts[item.id];
      return {
        key: `live:${question.request_id}:${item.id}`,
        round,
        item: { id: item.id, text, options: item.options, multi_select: item.multi_select ?? false },
        why,
        answer: draft && draftAnswer(round, text, draft),
        live: true,
      };
    }),
  };
};

// Answers saved after the last finished interview run, the one that wrote the memory; a hand edit is not a regeneration.
export const answersChangedSinceRun = (answers: InterviewAnswer[], trails: Trail[]): number => {
  const generated = trails.find((t) => t.state === "done" && t.ended_at !== null)?.ended_at;
  if (!generated) return answers.length;
  return answers.filter((a) => Date.parse(a.answered_at) > Date.parse(generated)).length;
};
