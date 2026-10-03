import { Pencil, SmilePlus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { EmojiPickerPopover } from "@/components/chat/EmojiPickerPopover";
import { useToggleReaction } from "@/hooks/ReactionHooks";
import type { Message } from "@/models/Chat";

interface MessageActionsProps {
  message: Message;
  // onEdit and onDelete are left out on someone else's message, which you may only react to.
  onEdit?: (() => void) | undefined;
  onDelete?: (() => void) | undefined;
}

// A pill pinned to the top-right corner of its bubble; the row's hover or focus reveals it, and an open picker holds it.
export const MessageActions = ({ message, onEdit, onDelete }: MessageActionsProps) => {
  const toggle = useToggleReaction(message);
  return (
    <div className="pointer-events-none absolute -top-3.5 -right-2 z-10 flex rounded-md border border-input bg-popover opacity-0 shadow-card transition-opacity duration-150 ease-standard group-focus-within:pointer-events-auto group-focus-within:opacity-100 group-hover:pointer-events-auto group-hover:opacity-100 has-[[data-state=open]]:pointer-events-auto has-[[data-state=open]]:opacity-100">
      <EmojiPickerPopover onPick={toggle} align="end">
        <Button size="icon" variant="ghost" className="size-6" aria-label="Add reaction">
          <SmilePlus className="size-3.5" aria-hidden />
        </Button>
      </EmojiPickerPopover>
      {onEdit && (
        <Button size="icon" variant="ghost" className="size-6" aria-label="Edit message" onClick={onEdit}>
          <Pencil className="size-3.5" aria-hidden />
        </Button>
      )}
      {onDelete && (
        <Button size="icon" variant="ghost" className="size-6" aria-label="Delete message" onClick={onDelete}>
          <Trash2 className="size-3.5" aria-hidden />
        </Button>
      )}
    </div>
  );
};
