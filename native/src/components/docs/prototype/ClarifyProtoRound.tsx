import { ChevronDown } from "lucide-react-native";
import { Pressable, View } from "react-native";
import { useShallow } from "zustand/react/shallow";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { ClarifyProtoNoteBox, ClarifyProtoNoteReply } from "@/components/docs/prototype/ClarifyProtoNote";
import { ClarifyProtoQuestionRow } from "@/components/docs/prototype/ClarifyProtoQuestionRow";
import { isDone, roundQuestions, useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";
import { cn } from "@/lib/utils";

interface ClarifyProtoRoundProps {
  n: number;
}

// One round as a foldable section of numbered rows; the newest open round ends with "Anything else?".
export const ClarifyProtoRound = ({ n }: ClarifyProtoRoundProps) => {
  const s = useClarifyProtoStore(
    useShallow((st) => ({
      answers: st.answers, skipped: st.skipped, openId: st.openId, folds: st.folds,
      rounds: st.rounds, phase: st.phase, variant: st.variant, toggleFold: st.toggleFold,
    })),
  );
  const questions = roundQuestions(n);
  const answered = questions.filter((q) => s.answers[q.id] !== undefined).length;
  const skipped = questions.filter((q) => s.skipped.includes(q.id)).length;
  const meta = skipped === 0 ? `${answered} of ${questions.length} answered` : `${answered} answered · ${skipped} skipped`;
  const newest = n === s.rounds;
  const live = newest && s.phase === "open";
  const hasPending = questions.some((q) => !isDone(s, q.id));
  const folded = s.folds[n] ?? (s.phase === "closed" || (!newest && !questions.some((q) => q.id === s.openId) && !hasPending));

  return (
    <View>
      <Pressable
        role="button"
        aria-expanded={!folded}
        onPress={() => s.toggleFold(n, folded)}
        android_ripple={{ borderless: false }}
        className="min-h-12 flex-row flex-wrap items-center justify-between gap-x-2 border-b border-border py-2"
      >
        <Text className="font-semibold">Round {n}</Text>
        <View className="flex-row items-center gap-1">
          <Text className="font-mono text-xs leading-snug text-muted-foreground">{meta}</Text>
          <Icon as={ChevronDown} size={16} className={cn("text-muted-foreground", folded && "-rotate-90")} />
        </View>
      </Pressable>
      {!folded &&
        questions.map((q, i) => (
          <ClarifyProtoQuestionRow key={q.id} item={q} number={i + 1} total={questions.length} first={n === 1 && i === 0} />
        ))}
      {!folded && live && s.variant !== "B" && <ClarifyProtoNoteBox round={n} />}
      {!live && <ClarifyProtoNoteReply round={n} />}
    </View>
  );
};
