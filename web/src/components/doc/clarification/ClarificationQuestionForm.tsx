import { QuestionForm } from "@/components/questions/QuestionForm";
import { useAnswerDocQuestion, useClearDocAnswer } from "@/hooks/DocClarificationHooks";
import { questionRow, type ClarificationQuestion } from "@/models/DocClarification";
import type { AnswerValue } from "@/models/Question";

interface ClarificationQuestionFormProps {
  question: ClarificationQuestion;
  progress: string;
  prevKey: string | null;
  nextKey: string | null;
  onMove: (key: string | null) => void;
}

// A doc question's open row: nothing picked up front, even the (Suggested) option; Next and Skip save, Clear makes it pending.
export const ClarificationQuestionForm = ({ question, progress, prevKey, nextKey, onMove }: ClarificationQuestionFormProps) => {
  const answer = useAnswerDocQuestion(question.doc_id);
  const clear = useClearDocAnswer(question.doc_id);

  const submit = (value: AnswerValue | null) => {
    const input = { selected: value?.selected ?? [], text: value?.text ?? "", skipped: value === null };
    answer.mutate({ questionId: question.id, answer: input }, { onSuccess: () => onMove(nextKey) });
  };

  return (
    <QuestionForm
      row={questionRow(question)}
      progress={progress}
      whyLabel="Why we're asking:"
      onBack={prevKey !== null ? () => onMove(prevKey) : undefined}
      onSubmit={submit}
      onClear={() => clear.mutate(question.id)}
      busy={answer.isPending || clear.isPending}
    />
  );
};
