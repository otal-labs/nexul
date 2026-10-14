import { questionRow, type ClarificationQuestion } from "@nexul/client-core/clarification";
import { answerLine, rowStatus, type ChecklistRowStatus } from "@nexul/client-core/questionChecklist";
import Check from "lucide-react-native/icons/check";
import Minus from "lucide-react-native/icons/minus";
import type { ReactNode } from "react";
import { Pressable, View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

interface ClarificationQuestionRowProps {
  question: ClarificationQuestion;
  number: number;
  open: boolean;
  readOnly: boolean;
  onToggle: () => void;
  // The question form, shown under the title while the row is open.
  children: ReactNode;
}

const Marker = ({ status, open, number }: { status: ChecklistRowStatus; open: boolean; number: number }) => (
  <View
    className={cn(
      "mt-px min-h-6 min-w-6 items-center justify-center rounded-full px-1",
      open && "bg-foreground",
      !open && status === "answered" && "bg-success",
      !open && status === "skipped" && "border border-border",
      !open && status === "pending" && "border border-border",
    )}
  >
    {!open && status === "answered" && <Icon as={Check} size={14} strokeWidth={3} className="text-background" />}
    {!open && status === "skipped" && <Icon as={Minus} size={14} strokeWidth={3} className="text-muted-foreground" />}
    {(open || status === "pending") && <Text className={cn("font-mono text-[11px]", open ? "text-background" : "text-muted-foreground")}>{number}</Text>}
  </View>
);

const STATUS_LABEL: Record<ChecklistRowStatus, string> = { answered: "Answered", skipped: "Skipped", pending: "Not answered" };

// One question as a hairline row: its marker, the question, and its answer on as many lines as it needs; the open row holds the form under it.
export const ClarificationQuestionRow = ({ question, number, open, readOnly, onToggle, children }: ClarificationQuestionRowProps) => {
  const row = questionRow(question);
  const status = rowStatus(row);
  return (
    <View className="border-b border-border">
      <Pressable
        role="button"
        aria-expanded={open}
        aria-label={`${STATUS_LABEL[status]}: ${question.question}`}
        disabled={readOnly}
        onPress={onToggle}
        className="min-h-12 flex-row items-start gap-3 py-3 active:bg-accent"
      >
        <Marker status={status} open={open} number={number} />
        <View className="min-w-0 flex-1">
          <Text className={cn("leading-snug", open ? "font-medium" : "text-muted-foreground")}>{question.question}</Text>
          {!open && status === "answered" && <Text className="mt-0.5 text-sm leading-snug text-muted-foreground/70">{answerLine(row.answer)}</Text>}
          {!open && status === "skipped" && <Text className="mt-0.5 text-sm leading-snug text-muted-foreground/70">Skipped</Text>}
        </View>
      </Pressable>
      {open && children}
    </View>
  );
};
