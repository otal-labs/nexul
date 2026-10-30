import type { LucideIcon } from "lucide-react";
import { memo } from "react";
import { Link } from "react-router";

import type { Conversation } from "@nexul/client-core/chat";

import { ChannelHeaderActions } from "@/components/chat/ChannelHeaderActions";
import { ConversationSwitcher } from "@/components/chat/ConversationSwitcher";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/stores/sidebarStore";

interface ConversationHeaderProps {
  conversation: Conversation;
  label: string;
  icon: LucideIcon;
  // A doc or ticket thread's title leads back to what it is about.
  subjectPath: string | undefined;
  /** In a call the title sits over the stage at wide widths, not over the chat column beside it. */
  inCall?: boolean;
}

// The header's hairline runs the panel's width; its name and actions sit in the message column.
const ConversationHeaderImpl = ({ conversation, label, icon: Icon, subjectPath, inCall = false }: ConversationHeaderProps) => {
  const railed = useSidebarStore((s) => s.collapsed);
  const wsPath = useWorkspacePath();
  const isChannel = conversation.kind === "channel" || conversation.kind === "voice_channel";

  return (
    <div className="h-12 shrink-0 border-b border-border">
      <div className={cn("mx-auto flex h-full w-full max-w-3xl items-center gap-2 px-3", inCall && "@[1000px]/thread:max-w-none @[1000px]/thread:px-6")}>
        {railed && <ConversationSwitcher workspaceId={conversation.workspace_id} conversationId={conversation.id} label={label} icon={Icon} />}
        {!railed && <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />}
        {!railed && subjectPath && (
          <Link to={wsPath(subjectPath)} dir="auto" title={label} className="min-w-0 truncate text-sm font-semibold underline-offset-4 hover:underline">
            {label}
          </Link>
        )}
        {!railed && !subjectPath && (
          <span dir="auto" title={label} className="min-w-0 truncate text-sm font-semibold">
            {label}
          </span>
        )}
        {isChannel && <ChannelHeaderActions channel={conversation} />}
      </div>
    </div>
  );
};

// Memoized so an arriving message does not render the header and its channel actions again.
export const ConversationHeader = memo(ConversationHeaderImpl);
