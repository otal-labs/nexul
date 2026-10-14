import { questionRow, type ClarificationRound } from "@nexul/client-core/clarification";
import { countLine } from "@nexul/client-core/questionChecklist";
import ChevronDown from "lucide-react-native/icons/chevron-down";
import { Pressable, View } from "react-native";

import { AnythingElseBox } from "@/components/docs/clarification/AnythingElseBox";
import { AnythingElseReply } from "@/components/docs/clarification/AnythingElseReply";
import { ClarificationQuestionForm } from "@/components/docs/clarification/ClarificationQuestionForm";
import { ClarificationQuestionRow } from "@/components/docs/clarification/ClarificationQuestionRow";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ClarificationRoundSectionProps {
  round: ClarificationRound;
  number: number;
  current: string | null;
  readOnly: boolean;
  folded: boolean;
  // The round still taking its "Anything else?"; an earlier one shows what was written and the reply instead.
  open: boolean;
  onToggleFold: () => void;
  onMove: (key: string | null) => void;
  around: (key: string) => { prevKey: string | null; nextKey: string | null };
}

// One round as a foldable section of numbered rows, ending with its "Anything else?".
export const ClarificationRoundSection = ({ round, number, current, readOnly, folded, open, onToggleFold, onMove, around }: ClarificationRoundSectionProps) => (
  <View>
    <Pressable
      role="button"
      aria-expanded={!folded}
      aria-label={`Round ${number}`}
      onPress={onToggleFold}
      className="min-h-12 flex-row flex-wrap items-center justify-between gap-x-2 border-b border-border py-2 active:bg-accent"
    >
      <View className="flex-row items-center gap-1">
        <Text role="heading" className="font-semibold">{`Round ${number}`}</Text>
        <Icon as={ChevronDown} size={16} className={cn("size-4 text-muted-foreground", folded && "-rotate-90")} />
      </View>
      <Text className="min-w-0 shrink font-mono text-xs leading-snug text-muted-foreground">{countLine(round.questions.map(questionRow))}</Text>
    </Pressable>
    {!folded &&
      round.questions.map((question, i) => (
        <ClarificationQuestionRow
          key={question.id}
          question={question}
          number={i + 1}
          open={question.id === current}
          readOnly={readOnly}
          onToggle={() => onMove(question.id === current ? null : question.id)}
        >
          <ClarificationQuestionForm question={question} progress={`Question ${i + 1} of ${round.questions.length}`} onMove={onMove} {...around(question.id)} />
        </ClarificationQuestionRow>
      ))}
    {!folded && open && <AnythingElseBox round={round} />}
    {!open && <AnythingElseReply round={round} />}
  </View>
);
