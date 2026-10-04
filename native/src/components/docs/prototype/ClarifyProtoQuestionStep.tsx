import { useState } from "react";
import { ScrollView, View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import type { AnswerValue } from "@/components/docs/prototype/ClarifyProtoData";
import { ClarifyProtoOptions } from "@/components/docs/prototype/ClarifyProtoOptions";
import { hasAnswer } from "@/components/docs/prototype/ClarifyProtoQuestionForm";
import { roundOf, roundQuestions, useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";

interface ClarifyProtoQuestionStepProps {
  id: string;
}

// One question as a whole screen: big question, why, options, and Back, Skip, Next pinned at the bottom.
export const ClarifyProtoQuestionStep = ({ id }: ClarifyProtoQuestionStepProps) => {
  const round = roundOf(id);
  const questions = roundQuestions(round);
  const index = questions.findIndex((q) => q.id === id);
  const item = questions[index];
  const saved = useClarifyProtoStore((s) => s.answers[id]);
  const [draft, setDraft] = useState<AnswerValue | undefined>(saved);

  const advance = (value: AnswerValue | null) => {
    const store = useClarifyProtoStore.getState();
    store.save(id, value);
    const next = useClarifyProtoStore.getState().openId;
    if (next && roundOf(next) === round) {
      store.setStep({ kind: "question", id: next });
      return;
    }
    if (round === store.rounds && store.phase === "open") {
      store.setStep({ kind: "note" });
      return;
    }
    store.setStep({ kind: "overview" });
  };

  if (!item) return null;
  const prev = questions[index - 1]?.id;

  return (
    <View className="flex-1">
      <ScrollView className="flex-1" contentContainerClassName="px-4 pb-6 pt-6" keyboardShouldPersistTaps="handled">
        <Text className="font-mono text-xs text-muted-foreground leading-snug">
          Question {index + 1} of {questions.length}
        </Text>
        <Text role="heading" className="mt-2 text-xl font-semibold tracking-tight leading-snug">
          {item.text}
        </Text>
        <Text className="mt-2 text-sm text-muted-foreground leading-snug">
          <Text className="text-sm text-foreground leading-snug">{"Why we're asking: "}</Text>
          {item.why}
        </Text>
        <View className="mt-5">
          <ClarifyProtoOptions item={item} draft={draft} onDraft={setDraft} />
        </View>
      </ScrollView>
      <View className="flex-row items-center gap-2 border-t border-border px-4 pt-3">
        {prev && (
          <Button variant="ghost" onPress={() => useClarifyProtoStore.getState().setStep({ kind: "question", id: prev })} className="h-12 px-4">
            <Text>Back</Text>
          </Button>
        )}
        <Button variant="ghost" onPress={() => advance(null)} className="h-12 px-4">
          <Text>Skip</Text>
        </Button>
        <Button disabled={!hasAnswer(draft)} onPress={() => advance(draft ?? null)} className="h-12 flex-1">
          <Text>Next</Text>
        </Button>
      </View>
    </View>
  );
};
