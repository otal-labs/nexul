import { MessagesSquare } from "lucide-react-native";

import { ConversationsFeed } from "@/components/chat/ConversationsFeed";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HandOff, useLoaderShown } from "@/components/HandOff";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ScreenHeader } from "@/components/ScreenHeader";
import { FieldScreen } from "@/components/FieldScreen";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useCurrentWorkspaceId, useSelectedWorkspace } from "@/hooks/WorkspaceHooks";

export const ChatListScreen = () => {
  const workspaceId = useCurrentWorkspaceId();
  const workspace = useSelectedWorkspace();
  const { data: conversations, error, isPending } = useFetchConversations(workspaceId);
  const waited = useLoaderShown(isPending);
  return (
    <FieldScreen>
      {!(conversations && conversations.length > 0) && <ScreenHeader eyebrow={workspace?.name} title="Chat" />}
      {isPending && <LoadingDisplay message="Loading conversations" />}
      {error && <ErrorDisplay error={error} />}
      {conversations && conversations.length === 0 && (
        <EmptyState icon={MessagesSquare} title="No conversations yet" message="Channels and direct messages started on the web show up here." />
      )}
      {conversations && conversations.length > 0 && (
        <HandOff after={waited}>
          <ConversationsFeed workspaceId={workspaceId} conversations={conversations} />
        </HandOff>
      )}
    </FieldScreen>
  );
};
