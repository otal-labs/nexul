import { useNavigate, useParams } from "react-router";

import { ConversationList } from "@/components/chat/ConversationList";
import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// Three panes: the app sidebar, this list, and the thread (ADR 0060). Below `sm` one pane shows at a time.
export const ChatPage = () => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { conversationId } = useParams();
  const navigate = useNavigate();
  const { data: conversations, error, isPending } = useFetchConversations(workspaceId);
  const { data: unread } = useFetchChatUnread(workspaceId);
  const selected = conversations?.find((c) => c.id === conversationId);

  return (
    <div className="flex h-screen">
      {isPending && <LoadingDisplay label="Loading chat…" />}
      {error && <ErrorDisplay error={error} title="Failed to load chat." />}
      {conversations && (
        <div
          className={cn(
            "min-h-0 sm:block sm:w-64 sm:shrink-0 sm:border-r sm:border-border lg:w-80",
            selected ? "hidden" : "block w-full",
          )}
        >
          <ConversationList
            workspaceId={workspaceId}
            conversations={conversations}
            unread={unread}
            selectedConversationId={selected?.id}
            onSelect={(id) => void navigate(`/chat/${id}`)}
          />
        </div>
      )}
      {conversations && (
        <div className={cn("min-w-0 flex-1 sm:flex sm:flex-col", selected ? "flex flex-col" : "hidden")}>
          {selected && (
            <ConversationThread workspaceId={workspaceId} conversation={selected} onBack={() => void navigate("/chat")} />
          )}
          {!selected && (
            <div className="flex h-full items-center justify-center">
              <NoDataDisplay message="Select a conversation" size="compact" />
            </div>
          )}
        </div>
      )}
    </div>
  );
};
