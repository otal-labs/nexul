import { useId, useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { QuestionStep } from "@/components/play/QuestionStep";
import { useSaveInterviewAnswer } from "@/hooks/MemoryHooks";
import { answerValue, type InterviewRow } from "@/models/InterviewAnswer";
import type { AnswerValue } from "@/models/Question";

interface InterviewQuestionFormProps {
  row: InterviewRow;
  projectId: string;
  progress: string;
  prevKey: string | null;
  nextKey: string | null;
  onMove: (key: string | null) => void;
}

const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The open row's body: the question's options flush under its title, with Back, Skip, and Next; Next and Skip save.
export const InterviewQuestionForm = ({ row, projectId, progress, prevKey, nextKey, onMove }: InterviewQuestionFormProps) => {
  const [draft, setDraft] = useState<AnswerValue | undefined>(answerValue(row.answer));
  const save = useSaveInterviewAnswer();
  const idPrefix = useId();
  const canNext = hasAnswer(draft) && !save.isPending;

  const submit = (skip: boolean) => {
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
        <Button variant="ghost" size="sm" disabled={save.isPending} onClick={() => submit(true)}>
          Skip
        </Button>
        <Button size="sm" loading={save.isPending} disabled={!canNext} onClick={() => submit(false)}>
          Next
        </Button>
      </div>
    </div>
  );
};
