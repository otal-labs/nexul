import { useRouter } from "expo-router";
import { Pressable, View } from "react-native";

import { HarnessProviderMark } from "@/components/chat/HarnessProviderMark";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { handoffStateDot, handoffStateLabel, type Handoff, type Message } from "@/models/Chat";

interface HandoffRowProps {
  message: Message;
  handoff: Handoff;
}

// One handed-off agent in the reply's sub-agents fold, opening its conversation.
export const HandoffRow = ({ message, handoff }: HandoffRowProps) => {
  const router = useRouter();
  return (
    <Pressable
      role="button"
      aria-label={`Open ${handoff.title}, ${handoffStateLabel(handoff.state)}`}
      onPress={() =>
        router.push({
          pathname: "/chat/handoff/[id]",
          params: { id: handoff.id, conversationId: message.conversation_id, messageId: message.id },
        })
      }
      className="min-h-11 flex-row items-center gap-2 border-t border-border px-3 active:bg-accent"
    >
      <HarnessProviderMark driver={handoff.driver} />
      <Text numberOfLines={1} className="shrink text-sm">
        {handoff.title}
      </Text>
      <Text numberOfLines={1} className="flex-1 font-mono text-xs text-muted-foreground">
        {handoff.model}
      </Text>
      <View className={cn("size-2 rounded-full", handoffStateDot(handoff.state))} />
    </Pressable>
  );
};
