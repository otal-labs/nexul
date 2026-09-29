import { Bot } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Badge } from "@/components/ui/badge";
import { MessageHeader } from "@/components/ui/message";
import type { Message as ChatMessage } from "@/models/Chat";
import { personLabel, type Person } from "@/models/Person";
import { cn } from "@/lib/utils";
import { formatClockTime, formatFullTime, formatRelativeTime } from "@/utils/TimeUtility";

export type MessageAlign = "start" | "end";

export const MessageRowAvatar = ({ isAgent, author }: { isAgent: boolean; author: Person }) => (
  <>
    {isAgent && (
      <span className="flex size-6 items-center justify-center rounded-full bg-accent text-accent-foreground">
        <Bot className="size-3.5" aria-hidden />
      </span>
    )}
    {!isAgent && <PersonAvatar login={author.login} src={author.avatar_url} className="size-6" />}
  </>
);

interface MessageRowHeaderProps {
  align: MessageAlign;
  message: ChatMessage;
  isAgent: boolean;
  author: Person;
}

export const MessageRowHeader = ({ align, message, isAgent, author }: MessageRowHeaderProps) => (
  <>
    {align === "start" && (
      <MessageHeader className="gap-2 px-1">
        <span className="truncate text-xs font-semibold text-foreground">{isAgent ? "Agent" : personLabel(author)}</span>
        {isAgent && (
          <Badge variant="outline" className="h-4 px-1 text-[9px] tracking-wide uppercase">
            App
          </Badge>
        )}
        {isAgent && <span className="text-[11px]">via {personLabel(author)}</span>}
        <span className="shrink-0 font-mono text-[11px]" title={formatFullTime(message.created_at)}>
          {formatRelativeTime(message.created_at)}
        </span>
        {message.edited_at && <span className="shrink-0 text-[11px]">(edited)</span>}
      </MessageHeader>
    )}
    {align === "end" && (
      <MessageHeader className="justify-end gap-2 px-1">
        <span className="shrink-0 font-mono text-[11px]" title={formatFullTime(message.created_at)}>
          {formatRelativeTime(message.created_at)}
        </span>
        {message.edited_at && <span className="shrink-0 text-[11px]">(edited)</span>}
      </MessageHeader>
    )}
  </>
);

// What a grouped message shows in place of its header; the row's hover and focus reveal it.
export const MessageContinuationTime = ({ createdAt, className }: { createdAt: string; className?: string }) => (
  <time dateTime={createdAt} title={formatFullTime(createdAt)} className={cn("font-mono text-[11px] whitespace-nowrap text-muted-foreground tabular-nums", className)}>
    {formatClockTime(createdAt)}
  </time>
);
