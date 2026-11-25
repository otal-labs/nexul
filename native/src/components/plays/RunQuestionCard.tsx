import { useState } from "react";
import { View } from "react-native";

import { answerText, hasAnswer, type AnswerValue } from "@nexul/client-core/question";

import { ClarificationOptions } from "@/components/docs/clarification/ClarificationOptions";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useAnswerRun, useCanActOnRun } from "@/hooks/TrailHooks";
import type { Trail } from "@/models/Trail";

// Answered in place while the run waits on its starter, and the run goes on in the same trail; afterwards, or for
// anyone else, it reads the question with what was answered.
export const RunQuestionCard = ({ trail }: { trail: Trail }) => {
  const canAnswer = useCanActOnRun(trail);
  const answer = useAnswerRun();
  const [drafts, setDrafts] = useState<Record<string, AnswerValue>>({});
  const question = trail.question;
  if (!question) return null;
  const open = trail.state === "waiting" && question.answer === undefined && canAnswer;
  const ready = question.questions.every((item) => hasAnswer(drafts[item.id]));

  return (
    <View className="gap-4 rounded-xl border border-border bg-card p-4">
      {question.questions.map((item) => (
        <View key={item.id} className="gap-1">
          {item.header && <Text className="font-mono text-xs text-muted-foreground">{item.header}</Text>}
          <Text className="text-body font-medium">{item.text}</Text>
          {open && <ClarificationOptions item={item} draft={drafts[item.id]} onDraft={(value) => setDrafts({ ...drafts, [item.id]: value })} />}
          {!open && question.answer && <Text className="text-row text-muted-foreground">{answerText(question.answer.answers[item.id]) || "No answer"}</Text>}
          {!open && !question.answer && <Text className="text-row text-muted-foreground">{"Waiting for the starter's answer"}</Text>}
        </View>
      ))}
      {answer.error && <ErrorDisplay error={answer.error} className="px-0" />}
      {open && (
        <Button disabled={!ready || answer.isPending} onPress={() => answer.mutate({ trailId: trail.id, answers: drafts })}>
          <Text>{answer.isPending ? "Sending…" : "Send answer"}</Text>
        </Button>
      )}
    </View>
  );
};
