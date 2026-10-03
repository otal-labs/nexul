import { useState, type ReactNode } from "react";

import { EmojiPicker, EmojiPickerContent, EmojiPickerFooter, EmojiPickerSearch } from "@/components/ui/emoji-picker";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

// The instance serves the emoji data itself (public/emojibase), so opening the picker never reaches a third party.
const emojibaseUrl = `${import.meta.env.BASE_URL}emojibase`;

interface EmojiPickerPopoverProps {
  onPick: (emoji: string) => void;
  // children is the trigger button the picker anchors to.
  children: ReactNode;
  align?: "start" | "end";
  // restoreFocus false leaves focus to onPick, so the composer keeps typing where the emoji landed.
  restoreFocus?: boolean;
}

export const EmojiPickerPopover = ({ onPick, children, align = "start", restoreFocus = true }: EmojiPickerPopoverProps) => {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align={align} className="w-fit p-0" onCloseAutoFocus={(e) => !restoreFocus && e.preventDefault()}>
        <EmojiPicker
          className="h-80"
          emojibaseUrl={emojibaseUrl}
          onEmojiSelect={({ emoji }) => {
            onPick(emoji);
            setOpen(false);
          }}
        >
          <EmojiPickerSearch placeholder="Search emoji…" aria-label="Search emoji" />
          <EmojiPickerContent />
          <EmojiPickerFooter />
        </EmojiPicker>
      </PopoverContent>
    </Popover>
  );
};
