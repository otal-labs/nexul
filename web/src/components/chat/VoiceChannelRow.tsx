import { MessageSquare, Volume2 } from "lucide-react";

import { conversationRowClass } from "@/components/chat/ConversationRow";
import { VoiceOccupantList } from "@/components/chat/VoiceOccupantAvatars";
import { cn } from "@/lib/utils";
import type { Conversation } from "@/models/Chat";
import type { VoiceOccupant } from "@/models/Voice";

interface VoiceChannelRowProps {
  conversation: Conversation;
  occupants: VoiceOccupant[];
  resolveLogin: (identity: string) => string;
  selected: boolean;
  /** Primary click (the row itself, Discord-style): joins the call — fetch token, connect. */
  onJoin: (conversationId: string) => void;
  /** Secondary affordance: opens the channel's text chat without joining voice. */
  onOpenText: (conversationId: string) => void;
}

// Two click targets: the row itself joins the call; the message icon opens the same conversation's text chat instead.
export const VoiceChannelRow = ({ conversation, occupants, resolveLogin, selected, onJoin, onOpenText }: VoiceChannelRowProps) => (
  <div className={cn(conversationRowClass(selected), "flex-col items-stretch gap-0 py-0 pr-0 pl-0")}>
    <div className="flex w-full items-center gap-2.5 py-2 pr-2 pl-3">
      <button
        type="button"
        onClick={() => onJoin(conversation.id)}
        aria-current={selected ? "true" : undefined}
        className="flex min-w-0 flex-1 items-center gap-2.5 text-left"
      >
        <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <Volume2 className="size-3.5" aria-hidden />
        </span>
        <span className="min-w-0 flex-1 truncate">{conversation.name}</span>
      </button>
      {/* Always visible, not hover-revealed — a hover-only affordance is unreachable on touch (320-414px law). */}
      <button
        type="button"
        onClick={() => onOpenText(conversation.id)}
        aria-label={`Open ${conversation.name} text chat`}
        className="shrink-0 rounded-md p-1.5 text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground"
      >
        <MessageSquare className="size-3.5" aria-hidden />
      </button>
    </div>
    {/* pl-3 + size-7 circle + gap-2.5 = the channel name's x — the avatars line up under it. */}
    <VoiceOccupantList occupants={occupants} resolveLogin={resolveLogin} className="pb-1 pl-[3.125rem]" />
  </div>
);
