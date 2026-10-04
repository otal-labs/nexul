import { InterviewDraftFrom } from "@/components/memory/InterviewDraftFrom";
import { QuestionForm } from "@/components/questions/QuestionForm";
import { useSaveInterviewAnswer } from "@/hooks/MemoryHooks";
import { recommendedDraft, SKIPPED_ANSWER, type InterviewRow } from "@/models/InterviewAnswer";
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

// The interview's open row: a drafted question opens on the draft with where it came from, so Next confirms it;
// otherwise its (Recommended) options are picked up front. Next and Skip save, or join the live round.
export const InterviewQuestionForm = ({ row, projectId, progress, prevKey, nextKey, onMove, onAnswer, pending = false }: InterviewQuestionFormProps) => {
  const save = useSaveInterviewAnswer();
  const proposed = row.draft?.state === "open" ? row.draft.value : undefined;

  const submit = (value: AnswerValue | null) => {
    if (onAnswer) {
      onMove(onAnswer(value ?? { text: SKIPPED_ANSWER }));
      return;
    }
    const input = {
      project_id: projectId,
      round: row.round,
      question: row.item.text,
      selected: value?.selected ?? [],
      text: value?.text ?? "",
      skip: value === null,
    };
    save.mutate(input, { onSuccess: () => onMove(nextKey) });
  };

  return (
    <QuestionForm
      row={row}
      progress={progress}
      whyLabel="Why I'm asking:"
      initialDraft={proposed ?? recommendedDraft(row.item)}
      onBack={prevKey !== null ? () => onMove(prevKey) : undefined}
      onSubmit={submit}
      busy={save.isPending || pending}
      drafted={row.draft?.value}
      note={row.draft && <InterviewDraftFrom draft={row.draft} />}
      multiline
    />
  );
};
