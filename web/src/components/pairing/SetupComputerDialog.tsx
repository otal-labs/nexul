import { useState, type ReactNode } from "react";

import { PAIR_DIALOG_FRAME, SETUP_DIALOG_FRAME, SETUP_LEAD } from "@/components/pairing/PairDialogFrame";
import { SetupDoneButton } from "@/components/pairing/SetupDoneButton";
import { SetupStep } from "@/components/pairing/SetupStep";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useSetupDraftStore } from "@/stores/setupDraftStore";
import type { Computer } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupComputerDialogProps {
  // Absent for a row that opens it from its own button or menu.
  trigger?: ReactNode;
  computer: Computer;
  // onClosed lets a link straight to a computer's Set up forget it.
  onClosed?: (() => void) | undefined;
  open?: boolean | undefined;
  onOpenChange?: ((open: boolean) => void) | undefined;
}

// A paired computer's setup: one turn per provider switched on, saved with Done.
export const SetupComputerDialog = ({ trigger, computer, onClosed, open: controlled, onOpenChange: notify }: SetupComputerDialogProps) => {
  const [uncontrolled, setOpen] = useState(false);
  const open = controlled ?? uncontrolled;
  // Every way out but Done is Cancel: Esc, the corner close, and the Cancel button drop the unsaved edits.
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    notify?.(next);
    useSetupDraftStore.getState().discard(computer.id);
    if (!next) onClosed?.();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {trigger && <DialogTrigger asChild>{trigger}</DialogTrigger>}
      <DialogContent
        className={cn(PAIR_DIALOG_FRAME, SETUP_DIALOG_FRAME)}
        // A stray click beside a long setup must not throw the step's choices away; Cancel and Esc close it on purpose.
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader className="gap-3 border-b border-border px-4 pt-5 pb-4 text-left sm:px-6">
          <DialogTitle className="pr-8">{`Set up ${computer.name}`}</DialogTitle>
          <DialogDescription>{SETUP_LEAD}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6">
          <SetupStep computer={computer} />
        </div>
        <div className="flex justify-end gap-2 border-t border-border px-4 py-3 sm:px-6">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </DialogClose>
          <SetupDoneButton computerId={computer.id} onDone={() => onOpenChange(false)} />
        </div>
      </DialogContent>
    </Dialog>
  );
};
