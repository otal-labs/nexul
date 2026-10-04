import { CircleCheck } from "lucide-react-native";
import { View } from "react-native";
import { useShallow } from "zustand/react/shallow";

import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";
import { ClarifyProtoRound } from "@/components/docs/prototype/ClarifyProtoRound";
import { pendingCount, useClarifyProtoStore, waitingLabel } from "@/components/docs/prototype/ClarifyProtoStore";
import { cn } from "@/lib/utils";

type StatusIcon = "waiting" | "done" | null;

interface StatusLine {
  icon: StatusIcon;
  label: string;
  detail: string;
}

// What the client sees of the clarification, as one line; nothing names who asks or how.
export const useStatusLine = (): StatusLine => {
  const s = useClarifyProtoStore(useShallow((st) => ({ answers: st.answers, skipped: st.skipped, phase: st.phase, rounds: st.rounds })));
  const pending = pendingCount(s);
  if (s.phase === "running") return { icon: null, label: "More questions are on the way", detail: "" };
  if (pending > 0) return { icon: "waiting", label: waitingLabel(pending), detail: `Round ${s.rounds}` };
  if (s.phase === "closed") return { icon: "done", label: "All answered", detail: `${s.rounds} ${s.rounds === 1 ? "round" : "rounds"}` };
  return { icon: "done", label: "All answered", detail: "Thanks, nothing is waiting on you." };
};

export const ClarifyProtoStatusIcon = ({ icon }: { icon: StatusIcon }) => (
  <>
    {icon === "waiting" && <View className="size-2 rounded-full bg-info" />}
    {icon === "done" && <Icon as={CircleCheck} size={16} className="text-success" />}
  </>
);

// The state line: the status icon, what is happening, a muted detail; wraps under itself at large font sizes.
export const ClarifyProtoStatus = ({ className }: { className?: string }) => {
  const { icon, label, detail } = useStatusLine();
  return (
    <View className={cn("flex-row flex-wrap items-center gap-x-2", className)}>
      <ClarifyProtoStatusIcon icon={icon} />
      <Text className={cn("text-sm leading-snug", icon ? "font-medium" : "text-muted-foreground")}>{label}</Text>
      {detail !== "" && <Text className="text-sm leading-snug text-muted-foreground">· {detail}</Text>}
    </View>
  );
};

// Every round so far, oldest first, under the "Questions" heading and its state line.
export const ClarifyProtoPanel = () => {
  const rounds = useClarifyProtoStore((s) => s.rounds);
  return (
    <View>
      <Text role="heading" className="text-lg font-semibold tracking-tight leading-snug">
        Questions
      </Text>
      <ClarifyProtoStatus className="mt-1" />
      <View className="mt-4 gap-5">
        {Array.from({ length: rounds }, (_, i) => (
          <ClarifyProtoRound key={i + 1} n={i + 1} />
        ))}
      </View>
    </View>
  );
};
