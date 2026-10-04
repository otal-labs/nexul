import { Check, Minus } from "lucide-react-native";
import { Pressable, View } from "react-native";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { answerLine, type ProtoQuestion } from "@/components/docs/prototype/ClarifyProtoData";
import { ClarifyProtoQuestionForm } from "@/components/docs/prototype/ClarifyProtoQuestionForm";
import { useClarifyProtoStore } from "@/components/docs/prototype/ClarifyProtoStore";
import { cn } from "@/lib/utils";

interface ClarifyProtoQuestionRowProps {
  item: ProtoQuestion;
  number: number;
  total: number;
  first: boolean;
}

type RowStatus = "answered" | "skipped" | "pending";

const rowStatus = (answered: boolean, skipped: boolean): RowStatus => {
  if (answered) return "answered";
  if (skipped) return "skipped";
  return "pending";
};

const Marker = ({ status, open, number }: { status: RowStatus; open: boolean; number: number }) => (
  <View
    className={cn(
      "mt-px min-h-6 min-w-6 items-center justify-center rounded-full px-1",
      open && "bg-foreground",
      !open && status === "answered" && "bg-success",
      !open && status === "skipped" && "bg-accent",
      !open && status === "pending" && "border border-border",
    )}
  >
    {!open && status === "answered" && <Icon as={Check} size={14} strokeWidth={3} className="text-background" />}
    {!open && status === "skipped" && <Icon as={Minus} size={14} strokeWidth={3} className="text-muted-foreground" />}
    {(open || status === "pending") && (
      <Text className={cn("font-mono text-[11px]", open ? "text-background" : "text-muted-foreground")}>{number}</Text>
    )}
  </View>
);

// One question as a hairline row: its marker, the question, and its one-line answer; the open row holds the form under it.
export const ClarifyProtoQuestionRow = ({ item, number, total, first }: ClarifyProtoQuestionRowProps) => {
  const answer = useClarifyProtoStore((s) => s.answers[item.id]);
  const skipped = useClarifyProtoStore((s) => s.skipped.includes(item.id));
  const variant = useClarifyProtoStore((s) => s.variant);
  const readOnly = useClarifyProtoStore((s) => s.phase === "closed");
  const open = useClarifyProtoStore((s) => s.openId === item.id && s.variant !== "B");
  const toggle = useClarifyProtoStore((s) => s.open);
  const setStep = useClarifyProtoStore((s) => s.setStep);
  const status = rowStatus(!!answer, skipped);

  const onPress = () => {
    if (variant === "B") {
      setStep({ kind: "question", id: item.id });
      return;
    }
    toggle(open ? null : item.id);
  };

  return (
    <View className="border-b border-border">
      <Pressable
        role="button"
        aria-expanded={open}
        disabled={readOnly}
        onPress={onPress}
        android_ripple={{ borderless: false }}
        className="min-h-12 flex-row items-start gap-3 py-3"
      >
        <Marker status={status} open={open} number={number} />
        <View className="min-w-0 flex-1">
          <Text className={cn("leading-snug", open ? "font-medium" : "text-muted-foreground")}>{item.text}</Text>
          {!open && status === "answered" && (
            <Text numberOfLines={1} className="mt-0.5 text-sm text-muted-foreground/70 leading-snug">
              {answerLine(answer)}
            </Text>
          )}
          {!open && status === "skipped" && <Text className="mt-0.5 text-sm text-muted-foreground/70 leading-snug">Skipped</Text>}
        </View>
      </Pressable>
      {open && <ClarifyProtoQuestionForm item={item} progress={`Question ${number} of ${total}`} first={first} />}
    </View>
  );
};
