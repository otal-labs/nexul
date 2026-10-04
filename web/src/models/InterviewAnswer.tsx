import type { InterviewQuestion } from "@/models/InterviewTemplate";
import { optionValue, type AnswerValue, type HarnessQuestion, type QuestionAnswers, type QuestionItem } from "@/models/Question";
import type { ChecklistRow } from "@/models/QuestionChecklist";
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

export interface InterviewRow extends ChecklistRow {
  round: number;
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
