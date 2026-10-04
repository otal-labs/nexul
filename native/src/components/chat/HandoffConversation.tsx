import { ScrollView, View } from "react-native";

import { HandoffStepRow } from "@/components/chat/HandoffStepRow";
import { MessageBody } from "@/components/chat/MessageBody";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import { handoffStateDot, handoffStateLabel, type Handoff } from "@/models/Chat";

interface HandoffConversationProps {
  handoff: Handoff;
}

export const HandoffConversation = ({ handoff }: HandoffConversationProps) => (
  <ScrollView contentContainerClassName="gap-4 p-4">
    <View className="flex-row items-center gap-2">
      <View className={cn("size-2 rounded-full", handoffStateDot(handoff.state))} />
      <Text className="text-sm">{handoffStateLabel(handoff.state)}</Text>
      {handoff.model !== "" && (
        <Text numberOfLines={1} className="shrink font-mono text-xs text-muted-foreground">
          {handoff.model}
        </Text>
      )}
    </View>
    <View className="gap-3 rounded-lg border border-border bg-card p-3">
      <MessageBody body={handoff.prompt} />
    </View>
    {handoff.steps.length > 0 && (
      <View>
        <Text className="font-mono text-[11px] uppercase text-muted-foreground">Steps · {handoff.steps.length}</Text>
        {handoff.steps.map((step, i) => (
          <HandoffStepRow key={i} step={step} />
        ))}
      </View>
    )}
    {handoff.state === "running" && handoff.steps.length === 0 && <LoadingDisplay message="Working…" />}
    {handoff.reply !== "" && (
      <View className="gap-3">
        <Text className="font-mono text-[11px] uppercase text-muted-foreground">Reply</Text>
        <MessageBody body={handoff.reply} />
      </View>
    )}
    {handoff.state !== "running" && handoff.reply === "" && (
      <Text variant="muted">{handoff.state === "left_running" ? "Still running in T3 Code." : "No reply came back."}</Text>
    )}
  </ScrollView>
);
