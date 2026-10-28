import { ChevronDown, type LucideIcon } from "lucide-react";
import { useState, type MouseEvent } from "react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ChatSidebarSection } from "@/components/sidebar/ChatSidebarSection";
import { UnreadBadge } from "@/components/UnreadBadge";
import { useFetchChatUnread, useFetchConversations } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import { listedUnread } from "@/models/Chat";

interface ConversationSwitcherProps {
  workspaceId: string | undefined;
  conversationId: string;
  label: string;
  icon: LucideIcon;
}

// The sidebar marks the open row with its sliding block, which the popover lacks; this draws the same tint and edge.
const openRowClass = cn(
  "[&_a[aria-current=page]]:bg-brand/12 dark:[&_a[aria-current=page]]:bg-brand/16",
  "[&_a[aria-current=page]]:before:absolute [&_a[aria-current=page]]:before:inset-y-2 [&_a[aria-current=page]]:before:left-0 [&_a[aria-current=page]]:before:w-0.5 [&_a[aria-current=page]]:before:rounded-full [&_a[aria-current=page]]:before:bg-brand [&_a[aria-current=page]]:before:content-['']",
);

// While the sidebar is the icon rail, the open conversation's name opens the sidebar's own conversation list.
export const ConversationSwitcher = ({ workspaceId, conversationId, label, icon: Icon }: ConversationSwitcherProps) => {
  const [open, setOpen] = useState(false);
  const { data: conversations } = useFetchConversations(workspaceId);
  const { data: unread } = useFetchChatUnread(workspaceId);
  const waiting = listedUnread(conversations ?? [], unread ?? {}, conversationId);
  // A pick is a link; a row's own menu or a section's + stays inside the list.
  const closeOnPick = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest("a")) setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={waiting > 0 ? `${label}, switch conversation, ${waiting} unread elsewhere` : `${label}, switch conversation`}
          className="-ml-1.5 flex h-8 min-w-0 items-center gap-2 rounded-md px-1.5 transition-colors duration-150 ease-standard hover:bg-accent"
        >
          <Icon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
          <span dir="auto" className="min-w-0 truncate text-sm font-semibold">
            {label}
          </span>
          <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <UnreadBadge count={waiting} />
        </button>
      </PopoverTrigger>
      <PopoverContent className={cn("max-h-[min(32rem,70dvh)] w-64 overflow-y-auto p-1.5", openRowClass)} onClickCapture={closeOnPick}>
        <ChatSidebarSection collapsed={false} />
      </PopoverContent>
    </Popover>
  );
};
