import { Smile } from "lucide-react";

import { EmojiPickerPopover } from "@/components/chat/EmojiPickerPopover";
import { Button } from "@/components/ui/button";

// The composer's emoji picker; the field keeps its caret while the picker has focus.
export const ComposerEmojiButton = ({ onPick }: { onPick: (emoji: string) => void }) => (
  <EmojiPickerPopover onPick={onPick} align="end" restoreFocus={false}>
    <Button type="button" size="icon" variant="ghost" aria-label="Add emoji" className="text-muted-foreground hover:text-foreground">
      <Smile className="size-4" aria-hidden />
    </Button>
  </EmojiPickerPopover>
);
