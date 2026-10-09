import { useRouter } from "expo-router";
import Bot from "lucide-react-native/icons/bot";
import { Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { handoffStateDot, handoffStateLabel, type Handoff, type Message } from "@/models/Chat";

interface HandoffPillProps {
  message: Message;
  handoff: Handoff;
}

export const HandoffPill = ({ message, handoff }: HandoffPillProps) => {
  const router = useRouter();
  const [mutedForeground] = useCSSVariable(["--color-muted-foreground"]);
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
      className="min-h-11 max-w-full flex-row items-center"
    >
      <View className="max-w-full flex-row items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 active:bg-accent">
        <Bot color={String(mutedForeground)} size={14} />
        {handoff.model !== "" && <Text className="font-mono text-[11px] text-muted-foreground">{handoff.model}</Text>}
        <Text numberOfLines={1} className="shrink text-xs">
          {handoff.title}
        </Text>
        <View className={cn("size-2 rounded-full", handoffStateDot(handoff.state))} />
      </View>
    </Pressable>
  );
};
