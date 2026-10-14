import { questionRow, type ClarificationQuestion } from "@nexul/client-core/clarification";
import { answerValue } from "@nexul/client-core/questionChecklist";
import { hasAnswer, type AnswerValue } from "@nexul/client-core/question";
import { useState } from "react";
import { View } from "react-native";

import { errorMessage } from "@/api/errors";
import { ClarificationOptions } from "@/components/docs/clarification/ClarificationOptions";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useAnswerDocQuestion, useClearDocAnswer } from "@/hooks/DocHooks";

interface ClarificationQuestionFormProps {
  question: ClarificationQuestion;
  progress: string;
  prevKey: string | null;
  nextKey: string | null;
  onMove: (key: string | null) => void;
}

// An open row's body: nothing picked up front, even the (Suggested) option; Next and Skip save, Clear makes it pending.
export const ClarificationQuestionForm = ({ question, progress, prevKey, nextKey, onMove }: ClarificationQuestionFormProps) => {
  const answer = useAnswerDocQuestion(question.doc_id);
  const clear = useClearDocAnswer(question.doc_id);
  const [draft, setDraft] = useState<AnswerValue | undefined>(answerValue(question));
  const busy = answer.isPending || clear.isPending;
  const answered = question.skipped || hasAnswer(question);
  const failure = answer.error ?? clear.error;

  const submit = (value: AnswerValue | null) =>
    answer.mutate(
      { questionId: question.id, answer: { selected: value?.selected ?? [], text: value?.text ?? "", skipped: value === null } },
      { onSuccess: () => onMove(nextKey) },
    );

  return (
    <View className="pb-4 pl-9">
      <Text className="font-mono text-xs leading-snug text-muted-foreground">{progress}</Text>
      {question.why !== "" && (
        <Text className="mt-1 text-sm leading-snug text-muted-foreground">
          <Text className="text-sm leading-snug">{"Why we're asking: "}</Text>
          {question.why}
        </Text>
      )}
      <ClarificationOptions item={questionRow(question).item} draft={draft} onDraft={setDraft} />
      {failure && <Text role="alert" className="mt-2 text-sm leading-snug text-destructive">{errorMessage(failure)}</Text>}
      <View className="mt-3 flex-row flex-wrap items-center justify-end gap-x-1">
        {answered && (
          <Button variant="ghost" size="sm" disabled={busy} onPress={() => clear.mutate(question.id, { onSuccess: () => setDraft(undefined) })} className="mr-auto">
            <Text>Clear answer</Text>
          </Button>
        )}
        {prevKey !== null && (
          <Button variant="ghost" size="sm" onPress={() => onMove(prevKey)}>
            <Text>Back</Text>
          </Button>
        )}
        <Button variant="ghost" size="sm" disabled={busy} onPress={() => submit(null)}>
          <Text>Skip</Text>
        </Button>
        <Button size="sm" disabled={busy || !hasAnswer(draft)} onPress={() => submit(draft ?? {})} className="px-5">
          <Text>Next</Text>
        </Button>
      </View>
    </View>
  );
};
