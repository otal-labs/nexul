import { ScrollView, View } from "react-native";

import { HandoffStepRow } from "@/components/chat/HandoffStepRow";
import { MessageBody } from "@/components/chat/MessageBody";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
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
    <View className="gap-3 rounded-xl border border-border bg-card p-4">
      <MessageBody body={handoff.prompt} />
    </View>
    {handoff.steps.length > 0 && (
      <View>
        <Microheader>{`Steps · ${handoff.steps.length}`}</Microheader>
        {handoff.steps.map((step, i) => (
          <HandoffStepRow key={i} step={step} />
        ))}
      </View>
    )}
    {handoff.state === "running" && handoff.steps.length === 0 && <LoadingDisplay message="Working" />}
    {handoff.reply !== "" && (
      <View className="gap-3">
        <Microheader>Reply</Microheader>
        <MessageBody body={handoff.reply} />
      </View>
    )}
    {handoff.state !== "running" && handoff.reply === "" && (
      <Text className="text-sm text-muted-foreground">{handoff.state === "left_running" ? "Still running in T3 Code." : "No reply came back."}</Text>
    )}
  </ScrollView>
);
