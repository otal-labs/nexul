import { useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { REPEAT_LABELS } from "@nexul/client-core/calendar";
import type { ScheduledMessage } from "@nexul/client-core/chat";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useCancelScheduledMessage, useSendScheduledMessageNow } from "@/hooks/ScheduledMessageHooks";
import { formatSendTime } from "@/lib/time";

// One scheduled message: when it posts and how often, its text, and Edit, Send now and Cancel; Cancel asks first.
export const ScheduledMessageRow = ({ scheduled }: { scheduled: ScheduledMessage }) => {
  const router = useRouter();
  const [confirming, setConfirming] = useState(false);
  const sendNow = useSendScheduledMessageNow();
  const cancel = useCancelScheduledMessage();
  const repeat = scheduled.repeats === "none" ? "" : ` · ${REPEAT_LABELS[scheduled.repeats]}`;
  const error = sendNow.error ?? cancel.error;
  const edit = () => router.push({ pathname: "/chat/schedule", params: { conversationId: scheduled.conversation_id, id: scheduled.id } });

  return (
    <View className="gap-2 rounded-md bg-surface-2 px-3 py-2.5">
      <Text className="font-mono text-micro text-muted-foreground">{formatSendTime(scheduled.send_at)}{repeat}</Text>
      <Text className="text-body leading-[22px]" numberOfLines={4}>{scheduled.body}</Text>
      {error && <ErrorDisplay error={error} className="px-0" />}
      {!confirming && (
        <View className="flex-row gap-2">
          <Button size="sm" variant="outline" className="flex-1" onPress={edit}>
            <Text>Edit</Text>
          </Button>
          <Button size="sm" variant="outline" className="flex-1" disabled={sendNow.isPending} onPress={() => sendNow.mutate(scheduled)}>
            <Text>{sendNow.isPending ? "Sending…" : "Send now"}</Text>
          </Button>
          <Button size="sm" variant="outline" className="flex-1" onPress={() => setConfirming(true)}>
            <Text>Cancel</Text>
          </Button>
        </View>
      )}
      {confirming && (
        <View className="flex-row gap-2">
          <Button size="sm" variant="destructive" className="flex-1" disabled={cancel.isPending} onPress={() => cancel.mutate(scheduled)}>
            <Text>{cancel.isPending ? "Cancelling…" : "Cancel message"}</Text>
          </Button>
          <Button size="sm" variant="outline" className="flex-1" disabled={cancel.isPending} onPress={() => setConfirming(false)}>
            <Text>Keep it</Text>
          </Button>
        </View>
      )}
    </View>
  );
};
