import { useId, useState, type KeyboardEvent, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { QuestionStep } from "@/components/questions/QuestionStep";
import { answerValue, type ChecklistRow } from "@nexul/client-core/questionChecklist";
import { hasAnswer, type AnswerValue } from "@nexul/client-core/question";

interface QuestionFormProps {
  row: ChecklistRow;
  progress: string;
  // Leads the row's why-line, in the voice of whoever asks.
  whyLabel: string;
  // The draft an unanswered row opens with.
  initialDraft?: AnswerValue | undefined;
  onBack?: (() => void) | undefined;
  // Told after each change the person makes to the draft whether it now holds an answer.
  onEdit?: ((drafted: boolean) => void) | undefined;
  // value null is a skip.
  onSubmit: (value: AnswerValue | null) => void;
  // Given, an answered or skipped row offers to make it pending again.
  onClear?: (() => void) | undefined;
  busy: boolean;
  // What a drafting run proposed: while the draft still holds it, Next reads Keep draft.
  drafted?: AnswerValue | undefined;
  // Passed to the step: a line under the hint, and a growing text box.
  note?: ReactNode;
  multiline?: boolean;
}

const sameAnswer = (a: AnswerValue | undefined, b: AnswerValue | undefined): boolean =>
  !!a && !!b && (a.text ?? "").trim() === (b.text ?? "").trim() && [...(a.selected ?? [])].sort().join("\n") === [...(b.selected ?? [])].sort().join("\n");

// The open row's body: the question's options flush under its title, with Back, Skip, and Next.
export const QuestionForm = ({
  row, progress, whyLabel, initialDraft, onBack, onEdit, onSubmit, onClear, busy, drafted, note, multiline = false,
}: QuestionFormProps) => {
  const [draft, setDraft] = useState<AnswerValue | undefined>(answerValue(row.answer) ?? initialDraft);
  const idPrefix = useId();
  const canNext = hasAnswer(draft) && !busy;
  const answered = row.answer !== undefined && (row.answer.skipped || hasAnswer(row.answer));

  const edit = (value: AnswerValue) => {
    setDraft(value);
    onEdit?.(hasAnswer(value));
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Enter" || !(e.target instanceof HTMLInputElement) || !canNext) return;
    e.preventDefault();
    onSubmit(draft ?? {});
  };

  return (
    <div onKeyDown={onKeyDown} className="animate-in fade-in-0 slide-in-from-top-1 pt-2 pr-2.5 pb-5 pl-10.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {row.why !== "" && (
        <p className="mt-1 text-sm text-pretty text-muted-foreground">
          <span className="text-foreground">{whyLabel} </span>
          {row.why}
        </p>
      )}
      <QuestionStep item={row.item} idPrefix={idPrefix} draft={draft} onDraft={edit} note={note} multiline={multiline} />
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
          {sameAnswer(draft, drafted) ? "Keep draft" : "Next"}
        </Button>
      </div>
    </div>
  );
};
