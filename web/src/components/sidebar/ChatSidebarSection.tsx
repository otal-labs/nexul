import { Hash, Users, Volume2 } from "lucide-react";
import { useNavigate } from "react-router";

import { VoiceOccupantList } from "@/components/chat/VoiceOccupantAvatars";
import { SidebarSectionHeader } from "@/components/sidebar/SidebarSectionHeader";
import { navLinkClass } from "@/components/SidebarNav";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
import { useNewConversationDialogs } from "@/hooks/useNewConversationDialogs";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useVoiceOccupancy } from "@/hooks/VoiceHooks";
import { cn } from "@/lib/utils";
import { conversationLabel, groupConversations, type Conversation, type DMLabelContext } from "@/models/Chat";
import type { VoiceOccupant } from "@/models/Voice";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface ChatSidebarRowProps {
  conversation: Conversation;
  unreadCount: number;
  dmCtx: DMLabelContext;
  onSelect: (conversationId: string) => void;
}

// Styled like ProjectTreeItem's NavLink rows, not ConversationRow's list styling — different surfaces.
const ChatSidebarRow = ({ conversation, unreadCount, dmCtx, onSelect }: ChatSidebarRowProps) => {
  const Icon = conversation.kind === "dm" ? Users : Hash;
  const label = conversationLabel(conversation, dmCtx);
  return (
    <button
      type="button"
      onClick={() => onSelect(conversation.id)}
      title={label}
      className={cn(navLinkClass({ isActive: false }), "w-full")}
    >
      <span className="flex w-8 shrink-0 justify-center">
        <Icon className="size-4" aria-hidden />
      </span>
      <span className="min-w-0 flex-1 truncate text-left">{label}</span>
      {unreadCount > 0 && (
        <span className="flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
          {unreadCount > 99 ? "99+" : unreadCount}
        </span>
      )}
    </button>
  );
};

interface VoiceChannelSidebarRowProps {
  conversation: Conversation;
  occupants: VoiceOccupant[];
  onJoin: (conversationId: string) => void;
}

// Same nav-row shell as ChatSidebarRow, but clicking joins the call, and lists occupants Discord-style.
const VoiceChannelSidebarRow = ({ conversation, occupants, onJoin }: VoiceChannelSidebarRowProps) => (
  <div>
    <button
      type="button"
      onClick={() => onJoin(conversation.id)}
      title={conversation.name}
      className={cn(navLinkClass({ isActive: false }), "w-full")}
    >
      <span className="flex w-8 shrink-0 justify-center">
        <Volume2 className="size-4" aria-hidden />
      </span>
      <span className="min-w-0 flex-1 truncate text-left">{conversation.name}</span>
    </button>
    {/* px-2.5 + w-8 icon column + gap-2.5 (navLinkClass geometry) = the channel name's x. */}
    <VoiceOccupantList occupants={occupants} className="pl-[3.25rem]" />
  </div>
);

interface ChatSidebarSectionProps {
  collapsed: boolean;
}

// Collapsed rail hides the list entirely (no room for labels); the Chat nav link still reaches the page.
export const ChatSidebarSection = ({ collapsed }: ChatSidebarSectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(collapsed ? undefined : workspaceId);
  const { data: unread } = useFetchChatUnread(workspaceId, !collapsed);
  const { data: me } = useFetchMe();
  const resolvePerson = usePersonLookup(workspaceId);
  const navigate = useNavigate();
  const occupancy = useVoiceOccupancy(!collapsed);
  const joinCall = useVoiceCallStore((s) => s.join);
  const canCreate = useAreaAccess()?.("newConversation") ?? false;
  // A new text channel or DM opens straight away; a new voice channel waits in the list to be joined.
  const { openNewChannel, openNewDM } = useNewConversationDialogs(workspaceId, (conversation) => {
    if (conversation.kind === "voice_channel") return;
    void navigate(`/chat/${conversation.id}`);
  });

  if (collapsed) return null;

  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolvePerson };
  const { channels, voiceChannels, dms } = groupConversations(conversations ?? []);

  // Joining voice does not open the thread; the call lives in the app-level VoiceDock instead.
  const handleJoinVoice = (conversationId: string) => {
    joinCall(conversationId);
  };

  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Channels"
          actionLabel="New channel"
          onAction={canCreate ? () => void openNewChannel(false) : undefined}
        />
          {channels.map((conversation) => (
            <ChatSidebarRow
              key={conversation.id}
              conversation={conversation}
              unreadCount={unread?.[conversation.id] ?? 0}
              dmCtx={dmCtx}
              onSelect={(id) => void navigate(`/chat/${id}`)}
            />
          ))}
      </div>
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Voice channels"
          actionLabel="New voice channel"
          onAction={canCreate ? () => void openNewChannel(true) : undefined}
        />
          {voiceChannels.map((conversation) => (
            <VoiceChannelSidebarRow
              key={conversation.id}
              conversation={conversation}
              occupants={occupancy[conversation.id] ?? []}
              onJoin={handleJoinVoice}
            />
          ))}
      </div>
      <div className="flex flex-col gap-0.5">
        <SidebarSectionHeader
          label="Direct messages"
          actionLabel="New direct message"
          onAction={canCreate ? () => void openNewDM() : undefined}
        />
          {dms.map((conversation) => (
            <ChatSidebarRow
              key={conversation.id}
              conversation={conversation}
              unreadCount={unread?.[conversation.id] ?? 0}
              dmCtx={dmCtx}
              onSelect={(id) => void navigate(`/chat/${id}`)}
            />
          ))}
      </div>
    </div>
  );
};
