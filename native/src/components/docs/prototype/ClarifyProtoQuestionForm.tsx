import { useState } from "react";
import { View } from "react-native";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { ClarifyProtoOptions } from "@/components/docs/prototype/ClarifyProtoOptions";
import type { AnswerValue, ProtoQuestion } from "@/components/docs/prototype/ClarifyProtoData";
import { useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";

interface ClarifyProtoQuestionFormProps {
  item: ProtoQuestion;
  progress: string;
  first: boolean;
}

export const hasAnswer = (draft: AnswerValue | undefined): boolean =>
  (draft?.text?.trim() ?? "") !== "" || (draft?.selected?.length ?? 0) > 0;

// The open row's body: where it sits in the round, why it's asked, the options, then Back, Skip, Next.
export const ClarifyProtoQuestionForm = ({ item, progress, first }: ClarifyProtoQuestionFormProps) => {
  const saved = useClarifyProtoStore((s) => s.answers[item.id]);
  const save = useClarifyProtoStore((s) => s.save);
  const back = useClarifyProtoStore((s) => s.back);
  const [draft, setDraft] = useState<AnswerValue | undefined>(saved);

  return (
    <View className="pb-4 pl-9">
      <Text className="font-mono text-xs text-muted-foreground leading-snug">{progress}</Text>
      <Text className="mt-1 text-sm text-muted-foreground leading-snug">
        <Text className="text-sm text-foreground leading-snug">{"Why we're asking: "}</Text>
        {item.why}
      </Text>
      <View className="mt-2">
        <ClarifyProtoOptions item={item} draft={draft} onDraft={setDraft} />
      </View>
      <View className="mt-3 flex-row items-center justify-end gap-1">
        {!first && (
          <Button variant="ghost" size="sm" onPress={() => back(item.id)}>
            <Text>Back</Text>
          </Button>
        )}
        <Button variant="ghost" size="sm" onPress={() => save(item.id, null)}>
          <Text>Skip</Text>
        </Button>
        <Button size="sm" disabled={!hasAnswer(draft)} onPress={() => save(item.id, draft ?? null)} className="px-5">
          <Text>Next</Text>
        </Button>
      </View>
    </View>
  );
};
