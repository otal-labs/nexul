import { Hash } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { cn } from "@/lib/utils";
import { conversationLabel, type Conversation, type DMLabelContext } from "@/models/Chat";

interface ConversationRowProps {
  conversation: Conversation;
  unreadCount: number;
  selected: boolean;
  onSelect: (conversationId: string) => void;
  dmCtx?: DMLabelContext;
}

export const conversationSectionLabelClass = "px-4 pt-3 pb-1 text-[11px] font-medium tracking-wide text-muted-foreground uppercase";

export const conversationRowClass = (selected: boolean) =>
  cn(
    "flex w-full items-center gap-2.5 border-l-2 py-2 pr-4 pl-3 text-left text-sm transition-colors duration-150 ease-standard",
    selected ? "border-foreground bg-accent" : "border-transparent hover:bg-accent/40",
  );

export const UnreadDot = ({ count }: { count: number }) => (
  <span aria-label={`${count} unread`} className="size-1.5 shrink-0 rounded-full bg-foreground" />
);

// A DM row leads with the other person's avatar; a channel leads with a hash in the same circle.
export const ConversationRow = ({ conversation, unreadCount, selected, onSelect, dmCtx }: ConversationRowProps) => {
  const label = conversationLabel(conversation, dmCtx);
  const isDM = conversation.kind === "dm";
  const other = conversation.participant_ids?.find((id) => id !== dmCtx?.currentUserId) ?? conversation.participant_ids?.[0];
  const avatarLogin = other && dmCtx ? dmCtx.resolveLogin(other) : label;
  return (
    <button
      type="button"
      onClick={() => onSelect(conversation.id)}
      aria-current={selected ? "true" : undefined}
      className={conversationRowClass(selected)}
    >
      {isDM && <PersonAvatar login={avatarLogin} className="size-7 text-[10px]" />}
      {!isDM && (
        <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Hash className="size-3.5" aria-hidden />
        </span>
      )}
      <span className={cn("min-w-0 flex-1 truncate", unreadCount > 0 && "font-medium")}>{label}</span>
      {unreadCount > 0 && <UnreadDot count={unreadCount} />}
    </button>
  );
};
