import { Navigate, useParams } from "react-router";

import { ChatPaneState } from "@/components/chat/ChatPaneState";
import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { defaultConversation } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The open conversation at full width; the app sidebar is the list (ADR 0093).
export const ChatPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { conversationId } = useParams();
  const wsPath = useWorkspacePath();
  const { data: conversations, error, isPending } = useFetchConversations(workspaceId);
  const selected = conversations?.find((c) => c.id === conversationId);
  const fallback = conversations && !conversationId ? defaultConversation(conversations) : undefined;

  return (
    <div className="flex h-full min-w-0 flex-col">
      {isPending && (
        <ChatPaneState>
          <LoadingDisplay label="Loading chat…" />
        </ChatPaneState>
      )}
      {error && (
        <ChatPaneState>
          <ErrorDisplay error={error} title="Failed to load chat." />
        </ChatPaneState>
      )}
      {fallback && <Navigate replace to={wsPath(`/chat/${fallback.id}`)} />}
      {conversations && !conversationId && !fallback && (
        <ChatPaneState>
          <NoDataDisplay message="No conversations yet" size="compact" />
        </ChatPaneState>
      )}
      {conversations && conversationId && !selected && (
        <ChatPaneState>
          <NoDataDisplay message="Conversation not found" size="compact" />
        </ChatPaneState>
      )}
      {selected && <ConversationThread workspaceId={workspaceId} conversation={selected} />}
    </div>
  );
};
