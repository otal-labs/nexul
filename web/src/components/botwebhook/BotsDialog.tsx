import { BotsSection } from "@/components/botwebhook/BotsSection";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import type { Conversation } from "@/models/Chat";

interface BotsDialogProps {
  conversation: Conversation;
  /** How the conversation is named in the title, a DM by its people. */
  label: string;
  open: boolean;
  onClose: () => void;
}

// A DM or a thread has no settings page, so its Bots section opens on its own.
export const BotsDialog = ({ conversation, label, open, onClose }: BotsDialogProps) => (
  <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
    <DialogContent className="flex max-h-[min(90dvh,52rem)] flex-col sm:max-w-md">
      <DialogHeader className="min-w-0 shrink-0 pr-8">
        <DialogTitle className="break-words">{label}</DialogTitle>
        <DialogDescription>The bots that post here.</DialogDescription>
      </DialogHeader>
      <div className="-mx-6 min-h-0 flex-1 overflow-y-auto px-6 py-0.5">
        <BotsSection conversation={conversation} />
      </div>
      <DialogFooter className="shrink-0">
        <DialogClose asChild>
          <Button size="sm">Done</Button>
        </DialogClose>
      </DialogFooter>
    </DialogContent>
  </Dialog>
);
