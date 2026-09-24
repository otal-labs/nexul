import { Bot } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Badge } from "@/components/ui/badge";
import { MessageHeader } from "@/components/ui/message";
import type { Message as ChatMessage } from "@/models/Chat";
import { formatRelativeTime } from "@/utils/TimeUtility";

export type MessageAlign = "start" | "end";

export const MessageRowAvatar = ({ isAgent, authorLogin }: { isAgent: boolean; authorLogin: string }) => (
  <>
    {isAgent && (
      <span className="flex size-6 items-center justify-center rounded-full bg-accent text-accent-foreground">
        <Bot className="size-3.5" aria-hidden />
      </span>
    )}
    {!isAgent && <PersonAvatar login={authorLogin} className="size-6" />}
  </>
);

interface MessageRowHeaderProps {
  align: MessageAlign;
  message: ChatMessage;
  isAgent: boolean;
  authorLogin: string;
}

export const MessageRowHeader = ({ align, message, isAgent, authorLogin }: MessageRowHeaderProps) => (
  <>
    {align === "start" && (
      <MessageHeader className="gap-2 px-1">
        <span className="truncate text-xs font-semibold text-foreground">{isAgent ? "Agent" : authorLogin}</span>
        {isAgent && (
          <Badge variant="outline" className="h-4 px-1 text-[9px] tracking-wide uppercase">
            App
          </Badge>
        )}
        {isAgent && <span className="text-[11px]">via {authorLogin}</span>}
        <span className="shrink-0 font-mono text-[11px]">{formatRelativeTime(message.created_at)}</span>
        {message.edited_at && <span className="shrink-0 text-[11px]">(edited)</span>}
      </MessageHeader>
    )}
    {align === "end" && (
      <MessageHeader className="justify-end gap-2 px-1">
        <span className="shrink-0 font-mono text-[11px]">{formatRelativeTime(message.created_at)}</span>
        {message.edited_at && <span className="shrink-0 text-[11px]">(edited)</span>}
      </MessageHeader>
    )}
  </>
);
