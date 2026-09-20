import { ConversationRow, conversationSectionLabelClass } from "@/components/chat/ConversationRow";
import { VoiceChannelRow } from "@/components/chat/VoiceChannelRow";
import { useChatAuthorLookup } from "@/hooks/ChatHooks";
import { useVoiceOccupancy } from "@/hooks/VoiceHooks";
import type { Conversation, DMLabelContext, UnreadCounts } from "@/models/Chat";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

interface ChatsSectionProps {
  workspaceId: string;
  channels: Conversation[];
  voiceChannels: Conversation[];
  dms: Conversation[];
  unread: UnreadCounts | undefined;
  selectedConversationId: string | undefined;
  onSelect: (conversationId: string) => void;
  dmCtx: DMLabelContext;
}

// Channels, voice channels, and DMs in one group; the row's leading mark says which is which.
export const ChatsSection = ({
  workspaceId,
  channels,
  voiceChannels,
  dms,
  unread,
  selectedConversationId,
  onSelect,
  dmCtx,
}: ChatsSectionProps) => {
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const occupancy = useVoiceOccupancy();
  const joinCall = useVoiceCallStore((s) => s.join);

  return (
    <div>
      <p className={conversationSectionLabelClass}>Chats</p>
      {channels.map((conversation) => (
        <ConversationRow
          key={conversation.id}
          conversation={conversation}
          unreadCount={unread?.[conversation.id] ?? 0}
          selected={conversation.id === selectedConversationId}
          onSelect={onSelect}
          dmCtx={dmCtx}
        />
      ))}
      {voiceChannels.map((conversation) => (
        <VoiceChannelRow
          key={conversation.id}
          conversation={conversation}
          occupants={occupancy[conversation.id] ?? []}
          resolveLogin={resolveLogin}
          selected={conversation.id === selectedConversationId}
          onJoin={joinCall}
          onOpenText={onSelect}
        />
      ))}
      {dms.map((conversation) => (
        <ConversationRow
          key={conversation.id}
          conversation={conversation}
          unreadCount={unread?.[conversation.id] ?? 0}
          selected={conversation.id === selectedConversationId}
          onSelect={onSelect}
          dmCtx={dmCtx}
        />
      ))}
    </div>
  );
};
