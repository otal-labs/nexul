import { Users } from "lucide-react";
import { useNavigate } from "react-router";

import { ChannelSidebarRow } from "@/components/sidebar/ChannelSidebarRow";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { DocThreadSidebarRow } from "@/components/sidebar/DocThreadSidebarRow";
import { SidebarSectionHeader } from "@/components/sidebar/SidebarSectionHeader";
import { VoiceChannelSidebarRow } from "@/components/sidebar/VoiceChannelSidebarRow";
import { sectionLabelClass } from "@/components/SidebarNav";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
import { useNewConversationDialogs } from "@/hooks/useNewConversationDialogs";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useVoiceOccupancy } from "@/hooks/VoiceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { conversationLabel, groupConversations, type DMLabelContext } from "@/models/Chat";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface ChatSidebarSectionProps {
  collapsed: boolean;
}

// Collapsed rail hides the list entirely (no room for labels); expand the sidebar to switch conversations.
export const ChatSidebarSection = ({ collapsed }: ChatSidebarSectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(collapsed ? undefined : workspaceId);
  const { data: unread } = useFetchChatUnread(workspaceId, !collapsed);
  const { data: me } = useFetchMe();
  const resolvePerson = usePersonLookup(workspaceId);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const occupancy = useVoiceOccupancy(!collapsed);
  const can = useAreaAccess();
  const canCreateChannel = can?.("editChannels") ?? false;
  const canCreateDM = can?.("newConversation") ?? false;
  // A new text channel or DM opens straight away; a new voice channel waits in the list to be joined.
  const { openNewChannel, openNewDM } = useNewConversationDialogs(workspaceId, (conversation) => {
    if (conversation.kind === "voice_channel") return;
    void navigate(wsPath(`/chat/${conversation.id}`));
  });

  if (collapsed) return null;

  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolvePerson };
  const { channels, voiceChannels, dms, docThreads } = groupConversations(conversations ?? []);

  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Channels"
          actionLabel="New channel"
          onAction={canCreateChannel ? () => void openNewChannel(false) : undefined}
        />
        {channels.map((conversation) => (
          <ChannelSidebarRow key={conversation.id} conversation={conversation} unreadCount={unread?.[conversation.id] ?? 0} />
        ))}
      </div>
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Voice channels"
          actionLabel="New voice channel"
          onAction={canCreateChannel ? () => void openNewChannel(true) : undefined}
        />
        {voiceChannels.map((conversation) => (
          <VoiceChannelSidebarRow
            key={conversation.id}
            conversation={conversation}
            occupants={occupancy[conversation.id] ?? []}
            unreadCount={unread?.[conversation.id] ?? 0}
          />
        ))}
      </div>
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Direct messages"
          actionLabel="New direct message"
          onAction={canCreateDM ? () => void openNewDM() : undefined}
        />
        {dms.map((conversation) => (
          <ChatSidebarRow
            key={conversation.id}
            conversationId={conversation.id}
            label={conversationLabel(conversation, dmCtx)}
            icon={Users}
            unreadCount={unread?.[conversation.id] ?? 0}
          />
        ))}
      </div>
      {/* No +: a thread is started from its doc, so the group only shows once one exists. */}
      {docThreads.length > 0 && (
        <div className="flex flex-col gap-0.5">
          <p className={sectionLabelClass}>Threads</p>
          {docThreads.map((conversation) => (
            <DocThreadSidebarRow key={conversation.id} conversation={conversation} unreadCount={unread?.[conversation.id] ?? 0} />
          ))}
        </div>
      )}
    </div>
  );
};
