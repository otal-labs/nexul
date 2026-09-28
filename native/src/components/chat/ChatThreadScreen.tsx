import { useLocalSearchParams } from "expo-router";
import { KeyboardAvoidingView, Platform, View } from "react-native";

import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatThreadTitle } from "@/components/chat/ChatThreadTitle";
import { MessageList } from "@/components/chat/MessageList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchMessages, useMarkThreadRead } from "@/hooks/ChatHooks";

export const ChatThreadScreen = () => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data: messages, error, isPending } = useFetchMessages(id);
  useMarkThreadRead(id, messages?.at(-1)?.id);
  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} className="flex-1 bg-background">
      <ChatThreadTitle conversationId={id} />
      {isPending && <LoadingDisplay />}
      {error && (
        <View className="flex-1 px-4">
          <ErrorDisplay error={error} />
        </View>
      )}
      {messages && messages.length === 0 && <PlaceholderScreen message="No messages yet." />}
      {messages && messages.length > 0 && <MessageList messages={messages} />}
      <ChatComposer conversationId={id} />
    </KeyboardAvoidingView>
  );
};
