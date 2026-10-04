import { View } from "react-native";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { ClarifyProtoDocBody, ClarifyProtoDocHeader } from "@/components/docs/prototype/ClarifyProtoDoc";
import { ClarifyProtoStepper } from "@/components/docs/prototype/ClarifyProtoStepper";
import { ClarifyProtoStatusIcon, useStatusLine } from "@/components/docs/prototype/ClarifyProtoStatus";
import { isDone, questionsUpTo, useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";

// The doc's one banner: the state line, Answer while questions wait, and a way to every round's answers.
const QuestionsBanner = () => {
  const { icon, label, detail } = useStatusLine();
  const s = useClarifyProtoStore(useShallow((st) => ({ answers: st.answers, skipped: st.skipped, rounds: st.rounds, setStep: st.setStep })));
  const waiting = icon === "waiting";
  const started = questionsUpTo(s.rounds).some((q) => isDone(s, q.id));
  const nextId = questionsUpTo(s.rounds).find((q) => !isDone(s, q.id))?.id;

  return (
    <View className="gap-3 rounded-lg border border-border bg-card p-4">
      <View>
        <View className="flex-row items-center gap-2">
          <ClarifyProtoStatusIcon icon={icon} />
          <Text className={waiting ? "font-semibold leading-snug" : "leading-snug text-muted-foreground"}>{label}</Text>
        </View>
        {detail !== "" && <Text className="text-sm leading-snug text-muted-foreground">{detail}</Text>}
      </View>
      <View className="flex-row flex-wrap gap-2">
        {waiting && nextId && (
          <Button onPress={() => s.setStep({ kind: "question", id: nextId })} className="h-12 grow basis-40">
            <Text>{started ? "Continue" : "Answer"}</Text>
          </Button>
        )}
        {started && (
          <Button variant="outline" onPress={() => s.setStep({ kind: "overview" })} className="h-12 grow basis-40">
            <Text>See answers</Text>
          </Button>
        )}
      </View>
    </View>
  );
};

// B: a banner on the doc opens a full-screen stepper, one question per screen.
export const ClarifyProtoVariantB = () => (
  <View className="gap-4">
    <ClarifyProtoDocHeader />
    <QuestionsBanner />
    <ClarifyProtoDocBody />
    <ClarifyProtoStepper />
  </View>
);
