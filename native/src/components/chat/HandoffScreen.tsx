import { Stack, useLocalSearchParams } from "expo-router";
import { View } from "react-native";

import { isNotFound } from "@/api/errors";
import { HandoffConversation } from "@/components/chat/HandoffConversation";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchMessages } from "@/hooks/ChatHooks";

type HandoffParams = { id: string; conversationId: string; messageId: string };

// Read from the thread's own messages, so a refetch that drops the reply's hand-offs (the reply was deleted) shows here too.
export const HandoffScreen = () => {
  const { id, conversationId, messageId } = useLocalSearchParams<HandoffParams>();
  const { data, error, isPending } = useFetchMessages(conversationId);
  const messages = isNotFound(error) ? undefined : data;
  const handoff = messages?.find((m) => m.id === messageId)?.handoffs?.find((h) => h.id === id);
  return (
    <View className="flex-1 bg-background">
      <Stack.Screen options={{ title: handoff?.title ?? "Hand-off" }} />
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} notFound="This conversation doesn't exist or was deleted." />}
      {messages && !handoff && <PlaceholderScreen message="This hand-off is no longer on its reply." />}
      {handoff && <HandoffConversation handoff={handoff} />}
    </View>
  );
};
