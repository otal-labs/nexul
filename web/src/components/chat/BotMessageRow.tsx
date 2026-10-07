import { BotAvatar } from "@/components/chat/BotAvatar";
import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedStack } from "@/components/chat/EmbedStack";
import { MessageActions } from "@/components/chat/MessageActions";
import { MessageReactions } from "@/components/chat/MessageReactions";
import { MessageContinuationTime } from "@/components/chat/MessageRowHeader";
import { Badge } from "@/components/ui/badge";
import { Message, MessageAvatar, MessageContent, MessageHeader } from "@/components/ui/message";
import { cn } from "@/lib/utils";
import type { Message as ChatMessage } from "@/models/Chat";
import { formatFullTime, formatRelativeTime } from "@/utils/TimeUtility";

const BotMessageHeader = ({ message }: { message: ChatMessage }) => (
  <MessageHeader className="gap-2 px-1">
    <span className="min-w-0 truncate text-xs font-semibold text-foreground">{message.author_name || "Bot"}</span>
    <Badge variant="outline" className="h-4 shrink-0 px-1 text-[9px] tracking-wide uppercase">
      Bot
    </Badge>
    <span className="shrink-0 font-mono text-[11px]" title={formatFullTime(message.created_at)}>
      {formatRelativeTime(message.created_at)}
    </span>
  </MessageHeader>
);

// Text and every embed share one bubble, widened when it carries an embed. Readers only react: bots are managed in settings.
const BotMessageBubble = ({ message }: { message: ChatMessage }) => {
  const embeds = message.embeds ?? [];
  return (
    <div
      data-slot="bubble"
      className={cn(
        "relative w-fit max-w-[75%] space-y-1.5 rounded-2xl rounded-bl-md bg-accent px-3 py-2 text-sm break-words text-accent-foreground",
        embeds.length > 0 && "w-full max-w-[min(75%,38rem)]",
      )}
    >
      <MessageActions message={message} />
      {message.body !== "" && (
        <DiscordMarkdown text={message.body} mentionHandles={(message.mentions ?? []).map((m) => m.handle)} className="space-y-0.5" />
      )}
      {embeds.length > 0 && <EmbedStack embeds={embeds} />}
    </div>
  );
};

// A bot's post under the name and avatar it posted with; the BOT tag keeps a post calling itself "GitHub" from passing for a person.
export const BotMessageRow = ({ message, continuation }: { message: ChatMessage; continuation: boolean }) => (
  <Message align="start" className="group px-3 py-0.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
    {!continuation && (
      <MessageAvatar className="size-6 self-start bg-transparent">
        <BotAvatar src={message.author_avatar_url} />
      </MessageAvatar>
    )}
    {continuation && (
      <MessageAvatar className="w-8 self-center overflow-visible bg-transparent">
        <MessageContinuationTime
          createdAt={message.created_at}
          className="opacity-0 transition-opacity duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100"
        />
      </MessageAvatar>
    )}
    <MessageContent>
      {!continuation && <BotMessageHeader message={message} />}
      <BotMessageBubble message={message} />
      <MessageReactions message={message} />
    </MessageContent>
  </Message>
);
