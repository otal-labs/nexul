import type { InterviewQuestion } from "@/models/InterviewTemplate";
import { optionValue, type AnswerValue, type HarnessQuestion, type QuestionAnswers, type QuestionItem } from "@/models/Question";

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

export interface InterviewRow {
  key: string;
  round: number;
  item: QuestionItem;
  why: string;
  answer: InterviewAnswer | undefined;
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

// The template's questions in template order, matched to their answers by trimmed text, then one section per stored round.
export const buildSections = (questions: InterviewQuestion[], answers: InterviewAnswer[]): InterviewSectionData[] => {
  const initial = answers.filter((a) => a.round === 0);
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
        answer: initial.find((a) => a.question === q.text.trim()),
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
  if (skipped === 0) return `${answered} of ${rows.length} answered`;
  return `${answered} answered · ${skipped} skipped`;
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

export const answerLine = (answer: InterviewAnswer | undefined): string =>
  answer ? [...answer.selected, answer.text].filter((part) => part !== "").join(" · ").replaceAll("\n", " · ") : "";

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
    skipped: selected.length === 0 && text.trim() === "", answered_by: "", answered_at: "",
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
