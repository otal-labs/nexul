import ChevronRight from "lucide-react-native/icons/chevron-right";
import { useState } from "react";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { formatElapsed } from "@nexul/client-core/agentSteps";
import { handoffsSpanSeconds, handoffTally, subAgentCount } from "@nexul/client-core/handoffs";

import { HandoffRow } from "@/components/chat/HandoffRow";
import { HarnessProviderMark } from "@/components/chat/HarnessProviderMark";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { Handoff, Message } from "@/models/Chat";

interface HandoffGroupProps {
  message: Message;
  handoffs: Handoff[];
}

const MAX_MARKS = 4;

// The agents an Agent reply handed work to as one fold, opening to a row per agent; the time shows once none runs.
export const HandoffGroup = ({ message, handoffs }: HandoffGroupProps) => {
  const [open, setOpen] = useState(false);
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  const running = handoffs.some((h) => h.state === "running");
  const seconds = running ? null : handoffsSpanSeconds(handoffs);
  const count = subAgentCount(handoffs.length);
  const tally = handoffTally(handoffs);
  return (
    <View className="mt-1 overflow-hidden rounded-lg border border-border bg-card">
      <Pressable
        role="button"
        aria-expanded={open}
        aria-label={`${count}, ${tally}${seconds === null ? "" : `, ${formatElapsed(seconds)}`}`}
        onPress={() => setOpen(!open)}
        className="min-h-11 flex-row items-center gap-2.5 px-3 py-1.5 active:bg-accent"
      >
        <View className="flex-row">
          {handoffs.slice(0, MAX_MARKS).map((h, i) => (
            <View key={h.id} className={cn("size-6 items-center justify-center rounded-full border-2 border-card bg-surface-2", i > 0 && "-ml-1")}>
              <HarnessProviderMark driver={h.driver} size={12} />
            </View>
          ))}
        </View>
        <View className="min-w-0 flex-1">
          <Text numberOfLines={1} className="text-xs font-semibold">
            {count}
          </Text>
          <Text numberOfLines={1} className={cn("text-xs", running ? "text-info" : "text-muted-foreground")}>
            {tally}
          </Text>
        </View>
        {seconds !== null && <Text className="font-mono text-xs text-muted-foreground">{formatElapsed(seconds)}</Text>}
        <ChevronRight color={String(muted)} size={14} style={{ transform: [{ rotate: open ? "90deg" : "0deg" }] }} />
      </Pressable>
      {open && handoffs.map((handoff) => <HandoffRow key={handoff.id} message={message} handoff={handoff} />)}
    </View>
  );
};
