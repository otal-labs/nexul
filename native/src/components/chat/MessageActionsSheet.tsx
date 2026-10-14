import { useLocalSearchParams, useRouter } from "expo-router";
import { useState } from "react";
import { View } from "react-native";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SheetTitle } from "@/components/SheetTitle";
import { Button } from "@/components/ui/button";
import { Text } from "@/components/ui/text";
import { useDeleteMessage, useFetchMessages } from "@/hooks/ChatHooks";
import { useMessageEditStore } from "@/stores/messageEditStore";

type MessageActionsParams = { conversationId: string; messageId: string; mode?: "delete" };

// Your own message: Edit hands its text to the composer, Delete asks first because the message is gone for everyone.
export const MessageActionsSheet = () => {
  const router = useRouter();
  const { conversationId, messageId, mode } = useLocalSearchParams<MessageActionsParams>();
  const { data: messages } = useFetchMessages(conversationId);
  const message = messages?.find((m) => m.id === messageId && !m.deleted_at);
  const [confirming, setConfirming] = useState(mode === "delete");
  const startEditing = useMessageEditStore((s) => s.start);
  const remove = useDeleteMessage(conversationId);

  const edit = () => {
    if (!message) return;
    startEditing({ conversationId, messageId, body: message.body });
    router.back();
  };

  return (
    <View className="gap-4 bg-popover px-5 pb-6">
      <SheetTitle title={confirming ? "Delete this message?" : "Your message"} className="px-0 pb-0" />
      {message && (
        <Text className="rounded-md bg-surface-2 px-3 py-2.5 text-[15px] leading-[22px]" numberOfLines={4}>
          {message.body}
        </Text>
      )}
      {!message && <Text className="text-[15px] leading-[22px] text-muted-foreground">This message is gone.</Text>}
      {message && confirming && (
        <Text className="text-[15px] leading-[22px] text-muted-foreground">
          It is removed for everyone in this conversation and cannot be brought back.
        </Text>
      )}
      {remove.error && <ErrorDisplay error={remove.error} className="px-0" />}
      {message && !confirming && (
        <>
          <Button onPress={edit}>
            <Text>Edit</Text>
          </Button>
          <Button variant="outline" onPress={() => setConfirming(true)}>
            <Text>Delete</Text>
          </Button>
        </>
      )}
      {message && confirming && (
        <>
          <Button variant="destructive" disabled={remove.isPending} onPress={() => remove.mutate(messageId, { onSuccess: () => router.back() })}>
            <Text>{remove.isPending ? "Deleting…" : "Delete message"}</Text>
          </Button>
          <Button variant="outline" disabled={remove.isPending} onPress={() => router.back()}>
            <Text>Keep message</Text>
          </Button>
        </>
      )}
      {!message && (
        <Button variant="outline" onPress={() => router.back()}>
          <Text>Close</Text>
        </Button>
      )}
    </View>
  );
};
