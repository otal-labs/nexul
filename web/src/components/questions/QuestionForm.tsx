import { useId, useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { QuestionStep } from "@/components/questions/QuestionStep";
import { answerValue, type ChecklistRow } from "@/models/QuestionChecklist";
import type { AnswerValue } from "@/models/Question";

interface QuestionFormProps {
  row: ChecklistRow;
  progress: string;
  // Leads the row's why-line, in the voice of whoever asks.
  whyLabel: string;
  // The draft an unanswered row opens with.
  initialDraft?: AnswerValue | undefined;
  onBack?: (() => void) | undefined;
  // value null is a skip.
  onSubmit: (value: AnswerValue | null) => void;
  // Given, an answered or skipped row offers to make it pending again.
  onClear?: (() => void) | undefined;
  busy: boolean;
}

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The open row's body: the question's options flush under its title, with Back, Skip, and Next.
export const QuestionForm = ({ row, progress, whyLabel, initialDraft, onBack, onSubmit, onClear, busy }: QuestionFormProps) => {
  const [draft, setDraft] = useState<AnswerValue | undefined>(answerValue(row.answer) ?? initialDraft);
  const idPrefix = useId();
  const canNext = hasAnswer(draft) && !busy;
  const answered = row.answer !== undefined && (row.answer.skipped || hasAnswer(row.answer));

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Enter" || !(e.target instanceof HTMLInputElement) || !canNext) return;
    e.preventDefault();
    onSubmit(draft ?? {});
  };

  return (
    <div onKeyDown={onKeyDown} className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {row.why !== "" && (
        <p className="mt-1 text-sm text-muted-foreground">
          <span className="text-foreground">{whyLabel} </span>
          {row.why}
        </p>
      )}
      <QuestionStep item={row.item} idPrefix={idPrefix} draft={draft} onDraft={setDraft} />
      <div className="mt-4 flex items-center justify-end gap-2">
        {onClear && answered && (
          <Button variant="ghost" size="sm" className="mr-auto" disabled={busy} onClick={onClear}>
            Clear answer
          </Button>
        )}
        {onBack && (
          <Button variant="ghost" size="sm" onClick={onBack}>
            Back
          </Button>
        )}
        <Button variant="ghost" size="sm" disabled={busy} onClick={() => onSubmit(null)}>
          Skip
        </Button>
        <Button size="sm" loading={busy} disabled={!canNext} onClick={() => onSubmit(draft ?? {})}>
          Next
        </Button>
      </div>
    </div>
  );
};
