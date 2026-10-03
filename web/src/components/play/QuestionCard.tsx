import { useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";

import { QuestionAnsweredList } from "@/components/play/QuestionAnsweredList";
import { QuestionStep, toggleOption } from "@/components/play/QuestionStep";
import { optionValue, type AnswerValue, type HarnessQuestion, type QuestionAnswer, type QuestionAnswers } from "@/models/Question";
import { cn } from "@/lib/utils";

interface QuestionCardProps {
  question: HarnessQuestion;
  // answer given means the card is read-only and shows it; no onSubmit means read-only without one.
  answer?: QuestionAnswer | undefined;
  onSubmit?: ((answers: QuestionAnswers) => void) | undefined;
  pending?: boolean;
  className?: string;
}

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// A stepped questionnaire: one question at a time, number keys pick, Enter confirms, the last step sends.
export const QuestionCard = ({ question, answer, onSubmit, pending = false, className }: QuestionCardProps) => {
  const [index, setIndex] = useState(0);
  const [drafts, setDrafts] = useState<QuestionAnswers>({});
  const items = question.questions;
  const item = items[index];
  const draft = item ? drafts[item.id] : undefined;
  const isLast = index >= items.length - 1;
  const canAdvance = hasAnswer(draft) && !pending;
  const readOnly = answer !== undefined || onSubmit === undefined;

  const setDraft = (value: AnswerValue) => {
    if (item) setDrafts((d) => ({ ...d, [item.id]: value }));
  };

  const advance = () => {
    if (!canAdvance) return;
    if (!isLast) {
      setIndex(index + 1);
      return;
    }
    onSubmit?.(drafts);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!item || readOnly) return;
    if (e.key === "Enter") {
      e.preventDefault();
      advance();
      return;
    }
    if (e.target instanceof HTMLInputElement && e.target.type === "text") return;
    const n = Number(e.key);
    const option = Number.isInteger(n) && n >= 1 && n <= 9 ? item.options[n - 1] : undefined;
    if (!option) return;
    e.preventDefault();
    setDraft(toggleOption(item, draft, optionValue(option)));
  };

  return (
    <div
      role="group"
      aria-label="Question from the Agent"
      tabIndex={0}
      onKeyDown={onKeyDown}
      className={cn("rounded-lg border border-border bg-card p-3 outline-none focus-visible:ring-2 focus-visible:ring-ring", className)}
    >
      {readOnly && <QuestionAnsweredList question={question} answer={answer} />}
      {!readOnly && item && (
        <div>
          <p className="font-mono text-[11px] text-info">
            Question {index + 1} of {items.length}
          </p>
          <h4 className="mt-1 text-sm font-medium text-balance">{item.text}</h4>
          <QuestionStep item={item} idPrefix={`question-${index}`} draft={draft} onDraft={setDraft} />
        </div>
      )}
      {!readOnly && item && (
        <div className="mt-3 flex items-center justify-end gap-2">
          {index > 0 && (
            <Button variant="ghost" size="sm" onClick={() => setIndex(index - 1)}>
              Back
            </Button>
          )}
          <Button size="sm" disabled={!canAdvance} onClick={advance}>
            {isLast ? "Send answer" : "Next"}
          </Button>
        </div>
      )}
    </div>
  );
};
