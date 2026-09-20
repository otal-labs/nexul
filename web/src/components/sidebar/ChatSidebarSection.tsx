import { Hash, Users, Volume2 } from "lucide-react";
import { useNavigate } from "react-router";

import { VoiceOccupantList } from "@/components/chat/VoiceOccupantAvatars";
import { navLinkClass, sectionLabelClass } from "@/components/SidebarNav";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useChatAuthorLookup, useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
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
  resolveLogin: (identity: string) => string;
  onJoin: (conversationId: string) => void;
}

// Same nav-row shell as ChatSidebarRow, but clicking joins the call, and lists occupants Discord-style.
const VoiceChannelSidebarRow = ({ conversation, occupants, resolveLogin, onJoin }: VoiceChannelSidebarRowProps) => (
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
    <VoiceOccupantList occupants={occupants} resolveLogin={resolveLogin} className="pl-[3.25rem]" />
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
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const navigate = useNavigate();
  const occupancy = useVoiceOccupancy(!collapsed);
  const joinCall = useVoiceCallStore((s) => s.join);

  if (collapsed) return null;
  if (!conversations || conversations.length === 0) return null;

  const dmCtx: DMLabelContext = { currentUserId: me?.user.id, resolveLogin };
  const { channels, voiceChannels, dms } = groupConversations(conversations);

  // Joining voice does not open the thread; the call lives in the app-level VoiceDock instead.
  const handleJoinVoice = (conversationId: string) => {
    joinCall(conversationId);
  };

  return (
    <div className="flex flex-col gap-0.5">
      {channels.length > 0 && (
        <div className="flex flex-col gap-0.5">
          <div className={sectionLabelClass}>Channels</div>
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
      )}
      {voiceChannels.length > 0 && (
        <div className="flex flex-col gap-0.5">
          <div className={sectionLabelClass}>Voice channels</div>
          {voiceChannels.map((conversation) => (
            <VoiceChannelSidebarRow
              key={conversation.id}
              conversation={conversation}
              occupants={occupancy[conversation.id] ?? []}
              resolveLogin={resolveLogin}
              onJoin={handleJoinVoice}
            />
          ))}
        </div>
      )}
      {dms.length > 0 && (
        <div className="flex flex-col gap-0.5">
          <div className={sectionLabelClass}>Direct messages</div>
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
      )}
    </div>
  );
};
