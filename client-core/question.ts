// Mirrors internal/harness Question and QuestionAnswer: one question with its options, and how it was answered.
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

export interface AnswerValue {
  selected?: string[];
  text?: string;
}

// What the harness wants back for an option; the label when it carries no explicit value.
export const optionValue = (option: QuestionOption): string => option.value ?? option.label;

// A pick replaces the typed text, and a single choice replaces the earlier pick; a multi-select flips the value in the set.
export const toggleOption = (item: QuestionItem, draft: AnswerValue | undefined, value: string): AnswerValue => {
  if (!item.multi_select) return { selected: [value] };
  const current = draft?.selected ?? [];
  return { selected: current.includes(value) ? current.filter((v) => v !== value) : [...current, value] };
};

export const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;
