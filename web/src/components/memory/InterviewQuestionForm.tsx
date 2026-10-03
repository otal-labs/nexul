import { useId, useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { QuestionStep } from "@/components/play/QuestionStep";
import { useSaveInterviewAnswer } from "@/hooks/MemoryHooks";
import { answerValue, recommendedDraft, SKIPPED_ANSWER, type InterviewRow } from "@/models/InterviewAnswer";
import type { AnswerValue } from "@/models/Question";

interface InterviewQuestionFormProps {
  row: InterviewRow;
  projectId: string;
  progress: string;
  prevKey: string | null;
  nextKey: string | null;
  onMove: (key: string | null) => void;
  // Given, the answer joins the run's live round instead of being saved, and returns the row to open next.
  onAnswer?: ((value: AnswerValue) => string | null) | undefined;
  pending?: boolean;
}

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The open row's body: the question's options flush under its title, with Back, Skip, and Next; Next and Skip save.
export const InterviewQuestionForm = ({ row, projectId, progress, prevKey, nextKey, onMove, onAnswer, pending = false }: InterviewQuestionFormProps) => {
  const [draft, setDraft] = useState<AnswerValue | undefined>(answerValue(row.answer) ?? recommendedDraft(row.item));
  const save = useSaveInterviewAnswer();
  const idPrefix = useId();
  const busy = save.isPending || pending;
  const canNext = hasAnswer(draft) && !busy;

  const submit = (skip: boolean) => {
    if (onAnswer) {
      onMove(onAnswer(skip ? { text: SKIPPED_ANSWER } : (draft ?? {})));
      return;
    }
    const input = {
      project_id: projectId,
      round: row.round,
      question: row.item.text,
      selected: skip ? [] : (draft?.selected ?? []),
      text: skip ? "" : (draft?.text ?? ""),
      skip,
    };
    save.mutate(input, { onSuccess: () => onMove(nextKey) });
  };

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Enter" || !(e.target instanceof HTMLInputElement) || !canNext) return;
    e.preventDefault();
    submit(false);
  };

  return (
    <div onKeyDown={onKeyDown} className="animate-in fade-in-0 slide-in-from-top-1 pr-1.5 pb-5 pl-9.5 duration-200 ease-out">
      <p className="font-mono text-[11px] text-muted-foreground">{progress}</p>
      {row.why !== "" && (
        <p className="mt-1 text-sm text-muted-foreground">
          <span className="text-foreground">Why I'm asking: </span>
          {row.why}
        </p>
      )}
      <QuestionStep item={row.item} idPrefix={idPrefix} draft={draft} onDraft={setDraft} />
      <div className="mt-4 flex items-center justify-end gap-2">
        {prevKey !== null && (
          <Button variant="ghost" size="sm" onClick={() => onMove(prevKey)}>
            Back
          </Button>
        )}
        <Button variant="ghost" size="sm" disabled={busy} onClick={() => submit(true)}>
          Skip
        </Button>
        <Button size="sm" loading={busy} disabled={!canNext} onClick={() => submit(false)}>
          Next
        </Button>
      </div>
    </div>
  );
};
