import { Volume2 } from "lucide-react";

import { VoiceOccupantList } from "@/components/chat/VoiceOccupantAvatars";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
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
  return (
    <div>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={conversation.name ?? "Voice channel"}
        icon={Volume2}
        unreadCount={unreadCount}
        onClick={() => void joinCall(conversation.id)}
      />
      {/* px-2.5 + w-8 icon column + gap-2.5 (navLinkClass geometry) = the channel name's x. */}
      <VoiceOccupantList occupants={occupants} className="pl-[3.25rem]" />
    </div>
  );
};
