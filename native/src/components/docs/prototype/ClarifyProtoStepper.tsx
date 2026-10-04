import { X } from "lucide-react-native";
import { KeyboardAvoidingView, Modal, ScrollView, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { DOC_TITLE } from "@/components/docs/prototype/ClarifyProtoData";
import { ClarifyProtoNoteBox } from "@/components/docs/prototype/ClarifyProtoNote";
import { ClarifyProtoQuestionStep } from "@/components/docs/prototype/ClarifyProtoQuestionStep";
import { ClarifyProtoPanel } from "@/components/docs/prototype/ClarifyProtoStatus";
import { pendingCount, questionsUpTo, isDone, roundOf, roundQuestions, useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";
import { cn } from "@/lib/utils";

// The segments of the round: done filled, the current one half, the rest a hairline.
const Progress = ({ round, currentId }: { round: number; currentId: string | null }) => {
  const done = useClarifyProtoStore(useShallow((s) => roundQuestions(round).map((q) => isDone(s, q.id))));
  return (
    <View className="flex-row gap-1 px-4">
      {roundQuestions(round).map((q, i) => (
        <View
          key={q.id}
          className={cn("h-1 flex-1 rounded-full", q.id === currentId && "bg-muted-foreground", q.id !== currentId && (done[i] ? "bg-foreground" : "bg-border"))}
        />
      ))}
    </View>
  );
};

const NoteStep = () => {
  const rounds = useClarifyProtoStore((s) => s.rounds);
  const setStep = useClarifyProtoStore((s) => s.setStep);
  const last = roundQuestions(rounds).at(-1)?.id;
  return (
    <View className="flex-1">
      <ScrollView className="flex-1" contentContainerClassName="px-4 pt-6" keyboardShouldPersistTaps="handled">
        <ClarifyProtoNoteBox round={rounds} large />
      </ScrollView>
      <View className="flex-row gap-2 border-t border-border px-4 pt-3">
        {last && (
          <Button variant="ghost" onPress={() => setStep({ kind: "question", id: last })} className="h-12 px-5">
            <Text>Back</Text>
          </Button>
        )}
        <Button onPress={() => setStep({ kind: "overview" })} className="h-12 flex-1">
          <Text>Done</Text>
        </Button>
      </View>
    </View>
  );
};

const Overview = () => {
  const setStep = useClarifyProtoStore((s) => s.setStep);
  const pending = useClarifyProtoStore((s) => pendingCount(s));
  const nextId = useClarifyProtoStore((s) => questionsUpTo(s.rounds).find((q) => !isDone(s, q.id))?.id);
  return (
    <View className="flex-1">
      <ScrollView className="flex-1" contentContainerClassName="px-4 pb-6 pt-4">
        <ClarifyProtoPanel />
      </ScrollView>
      <View className="border-t border-border px-4 pt-3">
        {pending > 0 && nextId && (
          <Button onPress={() => setStep({ kind: "question", id: nextId })} className="h-12">
            <Text>{pending === 1 ? "Answer 1 question" : `Answer ${pending} questions`}</Text>
          </Button>
        )}
        {pending === 0 && (
          <Button variant="outline" onPress={() => setStep(null)} className="h-12">
            <Text>Back to the doc</Text>
          </Button>
        )}
      </View>
    </View>
  );
};

// B's full-screen view over the doc and the tab bar; system back closes it like the cross.
export const ClarifyProtoStepper = () => {
  const step = useClarifyProtoStore((s) => s.step);
  const rounds = useClarifyProtoStore((s) => s.rounds);
  const setStep = useClarifyProtoStore((s) => s.setStep);
  const insets = useSafeAreaInsets();
  const round = step?.kind === "question" ? roundOf(step.id) : rounds;
  const title = step?.kind === "overview" ? DOC_TITLE : `Round ${round}`;

  return (
    <Modal visible={step !== null} animationType="slide" onRequestClose={() => setStep(null)} statusBarTranslucent navigationBarTranslucent>
      <KeyboardAvoidingView behavior="padding" className="flex-1 bg-background">
        <View className="flex-1" style={{ paddingTop: insets.top, paddingBottom: insets.bottom + 12 }}>
        <View className="min-h-14 flex-row items-center">
          <Button variant="ghost" aria-label="Close" onPress={() => setStep(null)} className="ml-1 h-12 w-12 px-0">
            <Icon as={X} size={22} />
          </Button>
          <Text numberOfLines={1} className="flex-1 text-center font-semibold leading-snug">
            {title}
          </Text>
          <View className="mr-1 w-12" />
        </View>
        {step?.kind !== "overview" && <Progress round={round} currentId={step?.kind === "question" ? step.id : null} />}
        {step?.kind === "question" && <ClarifyProtoQuestionStep key={step.id} id={step.id} />}
        {step?.kind === "note" && <NoteStep />}
        {step?.kind === "overview" && <Overview />}
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
};
