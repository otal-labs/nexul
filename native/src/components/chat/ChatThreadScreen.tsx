import { useLocalSearchParams } from "expo-router";
import { useHeaderHeight } from "expo-router/react-navigation";
import { KeyboardAvoidingView, View } from "react-native";

import { isNotFound } from "@/api/errors";
import { ChatComposer } from "@/components/chat/ChatComposer";
import { ChatThreadTitle } from "@/components/chat/ChatThreadTitle";
import { MessageList } from "@/components/chat/MessageList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchMessages, useMarkThreadRead } from "@/hooks/ChatHooks";

export const ChatThreadScreen = () => {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, error, isPending } = useFetchMessages(id);
  // A refetch keeps the last messages; a thread that is gone (deleted, or a private channel the viewer left) drops them.
  const gone = isNotFound(error);
  const messages = gone ? undefined : data;
  useMarkThreadRead(id, messages?.at(-1)?.id);
  // Edge-to-edge Android never resizes the window for the keyboard, and this view's frame starts below the header.
  const headerHeight = useHeaderHeight();
  return (
    <KeyboardAvoidingView behavior="padding" keyboardVerticalOffset={headerHeight} className="flex-1 bg-background">
      <ChatThreadTitle conversationId={id} />
      {isPending && <LoadingDisplay />}
      {error && (
        <View className="flex-1 px-4">
          <ErrorDisplay error={error} className="px-0" notFound="This conversation doesn't exist or was deleted." />
        </View>
      )}
      {messages && messages.length === 0 && <PlaceholderScreen message="No messages yet." />}
      {messages && messages.length > 0 && <MessageList messages={messages} />}
      {!gone && <ChatComposer conversationId={id} />}
    </KeyboardAvoidingView>
  );
};
