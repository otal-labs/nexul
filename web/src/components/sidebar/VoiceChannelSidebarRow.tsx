import { Volume2 } from "lucide-react";

import { VoiceOccupantList } from "@/components/chat/VoiceOccupantAvatars";
import { RowActions } from "@/components/listpane/RowActions";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useChannelRowActions } from "@/hooks/useChannelRowActions";
import type { Conversation } from "@/models/Chat";
import type { VoiceOccupant } from "@/models/Voice";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

interface VoiceChannelSidebarRowProps {
  conversation: Conversation;
  occupants: VoiceOccupant[];
  unreadCount: number;
}

// One click joins the call and opens the channel's text chat; joining an already-joined channel is a no-op.
export const VoiceChannelSidebarRow = ({ conversation, occupants, unreadCount }: VoiceChannelSidebarRowProps) => {
  const joinCall = useVoiceCallStore((s) => s.join);
  const { onRename, onDelete } = useChannelRowActions(conversation);
  const label = conversation.name ?? "Voice channel";
  return (
    <div className="flex flex-col gap-0.5">
      <ChatSidebarRow
        conversationId={conversation.id}
        label={label}
        icon={Volume2}
        unreadCount={unreadCount}
        onClick={() => void joinCall(conversation.id)}
        actions={(onRename || onDelete) && <RowActions itemLabel={label} onRename={onRename} onDelete={onDelete} />}
      />
      <VoiceOccupantList occupants={occupants} />
    </div>
  );
};
