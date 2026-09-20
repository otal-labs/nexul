// Mirrors internal/harness Question and QuestionAnswer: what a turn stopped to ask, and how the user answered.
export interface QuestionOption {
  label: string;
  description?: string;
  value?: string;
}

export interface QuestionItem {
  id: string;
  text: string;
  header?: string;
  options: QuestionOption[];
  multi_select?: boolean;
}

export interface HarnessQuestion {
  request_id: string;
  questions: QuestionItem[];
}

export interface AnswerValue {
  selected?: string[];
  text?: string;
}

export type QuestionAnswers = Record<string, AnswerValue>;

export interface QuestionAnswer {
  answers: QuestionAnswers;
}

// What the harness wants back for an option; the label when it carries no explicit value.
export const optionValue = (option: QuestionOption): string => option.value ?? option.label;

// One answer as a line: the free text, else the chosen values.
export const answerText = (answer: AnswerValue | undefined): string => {
  if (!answer) return "";
  if (answer.text) return answer.text;
  return (answer.selected ?? []).join(", ");
};

const QUESTION_FENCE = "```nexul-question\n";

// The Agent posts a question as its own message with the question JSON fenced; anything else is a plain message.
export const parseQuestionMessage = (body: string): HarnessQuestion | null => {
  if (!body.startsWith(QUESTION_FENCE) || !body.trimEnd().endsWith("```")) return null;
  const json = body.slice(QUESTION_FENCE.length, body.trimEnd().length - 3);
  try {
    const parsed: unknown = JSON.parse(json);
    if (typeof parsed !== "object" || parsed === null || !("request_id" in parsed) || !("questions" in parsed)) return null;
    return parsed as HarnessQuestion;
  } catch {
    return null;
  }
};
