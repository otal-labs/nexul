import { View } from "react-native";

import { ConversationsFeed } from "@/components/chat/ConversationsFeed";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";

export const ChatListScreen = () => {
  const workspaceId = useCurrentWorkspaceId();
  const { data: conversations, error, isPending } = useFetchConversations(workspaceId);
  return (
    <View className="flex-1 bg-background">
      {isPending && <LoadingDisplay />}
      {error && (
        <View className="px-4">
          <ErrorDisplay error={error} />
        </View>
      )}
      {conversations && conversations.length === 0 && <PlaceholderScreen message="No conversations yet." />}
      {conversations && conversations.length > 0 && (
        <ConversationsFeed workspaceId={workspaceId} conversations={conversations} />
      )}
    </View>
  );
};
