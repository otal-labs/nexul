import { useState } from "react";

import { ChatsSection } from "@/components/chat/ChatsSection";
import { conversationSectionLabelClass } from "@/components/chat/ConversationRow";
import { DocThreadRow } from "@/components/chat/DocThreadRow";
import { NewConversationMenu } from "@/components/chat/NewConversationMenu";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Input } from "@/components/ui/input";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useChatAuthorLookup } from "@/hooks/ChatHooks";
import { conversationLabel, groupConversations, type Conversation, type DMLabelContext, type UnreadCounts } from "@/models/Chat";

interface ConversationListProps {
  workspaceId: string;
  conversations: Conversation[];
  unread: UnreadCounts | undefined;
  selectedConversationId: string | undefined;
  onSelect: (conversationId: string) => void;
}

// Two groups (ADR 0060): chats are places to talk, threads are the discussion attached to a doc.
export const ConversationList = ({ workspaceId, conversations, unread, selectedConversationId, onSelect }: ConversationListProps) => {
  const [search, setSearch] = useState("");
  const { data: me } = useFetchMe();
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolveLogin };

  const query = search.trim().toLowerCase();
  const filtered =
    query === "" ? conversations : conversations.filter((c) => conversationLabel(c, dmCtx).toLowerCase().includes(query));
  const { channels, voiceChannels, dms, docThreads } = groupConversations(filtered);
  const hasChats = channels.length + voiceChannels.length + dms.length > 0;

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-border pr-2 pl-4">
        <h1 className="text-sm font-semibold">Chat</h1>
        <NewConversationMenu workspaceId={workspaceId} onCreated={onSelect} />
      </div>
      <div className="px-3 pt-3">
        <Input
          aria-label="Search conversations"
          placeholder="Search…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-8 text-xs"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto pb-2">
        {conversations.length === 0 && <NoDataDisplay message="No conversations yet" size="compact" />}
        {conversations.length > 0 && (
          <ChatsSection
            workspaceId={workspaceId}
            channels={channels}
            voiceChannels={voiceChannels}
            dms={dms}
            unread={unread}
            selectedConversationId={selectedConversationId}
            onSelect={onSelect}
            dmCtx={dmCtx}
          />
        )}
        {conversations.length > 0 && !hasChats && <p className="px-4 py-1 text-xs text-muted-foreground">No chats yet</p>}
        {conversations.length > 0 && (
          <div>
            <p className={conversationSectionLabelClass}>Threads</p>
            {docThreads.map((conversation) => (
              <DocThreadRow
                key={conversation.id}
                conversation={conversation}
                unreadCount={unread?.[conversation.id] ?? 0}
                selected={conversation.id === selectedConversationId}
                onSelect={onSelect}
              />
            ))}
            {docThreads.length === 0 && <p className="px-4 py-1 text-xs text-muted-foreground">No threads yet</p>}
          </div>
        )}
      </div>
    </div>
  );
};
