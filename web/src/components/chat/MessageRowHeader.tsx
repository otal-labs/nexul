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
    {!isAgent && <PersonAvatar login={author.login} name={personLabel(author)} src={author.avatar_url} className="size-8 text-xs" />}
  </>
);

interface MessageRowHeaderProps {
  message: ChatMessage;
  isAgent: boolean;
  author: Person;
}

export const MessageRowHeader = ({ message, isAgent, author }: MessageRowHeaderProps) => (
  <MessageHeader className="gap-2 px-0">
    <span className="truncate text-row font-semibold text-foreground">{isAgent ? "Agent" : personLabel(author)}</span>
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
);

const revealOnHover = "opacity-0 transition-opacity duration-150 ease-standard group-hover:opacity-100 group-has-[:focus]:opacity-100 [@media(hover:none)]:group-focus:opacity-100";

// Your message's time sits left of its bubble on the last line (mb-0 undoes the bubble's space-y), one grammar with the time beside a name; grouped, it waits for a hover.
export const OwnMessageMeta = ({ message, revealed }: { message: ChatMessage; revealed: boolean }) => (
  <span className={cn("absolute right-full bottom-2 mr-2 mb-0 flex items-center gap-1.5 text-xs leading-5 whitespace-nowrap text-muted-foreground", !revealed && revealOnHover)}>
    {message.via && <span>via {message.via}</span>}
    {message.edited_at && <span>(edited)</span>}
    <time dateTime={message.created_at} title={formatFullTime(message.created_at)} className="font-mono tabular-nums">
      {formatClockTime(message.created_at)}
    </time>
  </span>
);

// What a grouped message shows in place of its header; the row's hover or focus reveals it (on touch, a tap on the row).
export const MessageContinuationTime = ({ createdAt, className }: { createdAt: string; className?: string }) => (
  <time dateTime={createdAt} title={formatFullTime(createdAt)} className={cn("font-mono text-xs whitespace-nowrap text-muted-foreground tabular-nums", revealOnHover, className)}>
    {formatClockTime(createdAt)}
  </time>
);
