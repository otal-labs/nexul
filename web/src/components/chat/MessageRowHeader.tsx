import { Bot } from "lucide-react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { microheaderClass } from "@/components/Microheader";
import { Badge } from "@/components/ui/badge";
import { MessageHeader } from "@/components/ui/message";
import type { Message as ChatMessage } from "@/models/Chat";
import { cn } from "@/lib/utils";
import { formatClockTime, formatFullTime } from "@/utils/TimeUtility";

export type MessageAlign = "start" | "end";

export const MessageRowAvatar = ({ isAgent, author }: { isAgent: boolean; author: Person }) => (
  <>
    {isAgent && (
      <span className="flex size-8 items-center justify-center rounded-full bg-accent text-accent-foreground">
        <Bot className="size-4" aria-hidden />
      </span>
    )}
    {!isAgent && <PersonAvatar login={author.login} src={author.avatar_url} className="size-8 text-xs" />}
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
        <span className="truncate text-[13px] font-semibold text-foreground">{isAgent ? "Agent" : personLabel(author)}</span>
        {isAgent && (
          <Badge variant="outline" className={cn(microheaderClass, "h-4 shrink-0 px-1")}>
            App
          </Badge>
        )}
        {isAgent && <span className="min-w-0 truncate">via {personLabel(author)}</span>}
        <span className="shrink-0 font-mono" title={formatFullTime(message.created_at)}>
          {formatClockTime(message.created_at)}
        </span>
        {message.via && <span className="shrink-0">via {message.via}</span>}
        {message.edited_at && <span className="shrink-0">(edited)</span>}
      </MessageHeader>
    )}
    {align === "end" && (
      <MessageHeader className="justify-end gap-2 px-1">
        <span className="shrink-0 font-mono" title={formatFullTime(message.created_at)}>
          {formatClockTime(message.created_at)}
        </span>
        {message.via && <span className="shrink-0">via {message.via}</span>}
        {message.edited_at && <span className="shrink-0">(edited)</span>}
      </MessageHeader>
    )}
  </>
);

// What a grouped message shows in place of its header; the row's hover and focus reveal it.
export const MessageContinuationTime = ({ createdAt, className }: { createdAt: string; className?: string }) => (
  <time dateTime={createdAt} title={formatFullTime(createdAt)} className={cn("font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums", className)}>
    {formatClockTime(createdAt)}
  </time>
);
