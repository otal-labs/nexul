import { useState } from "react";

import { TerminalScreen } from "@/components/terminal/TerminalScreen";
import { TerminalStatusMarker } from "@/components/terminal/TerminalStatusMarker";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import type { TerminalState } from "@/lib/terminal";

interface TerminalDialogProps {
  // Where it runs, the computer's or the runner's name.
  title: string;
  socketPath: string;
  onClose: () => void;
}

// A live shell on a computer or a machine's runner. Escape belongs to the shell; a clean exit closes it, a failed one
// stays open so its code can be read.
export const TerminalDialog = ({ title, socketPath, onClose }: TerminalDialogProps) => {
  const [state, setState] = useState<TerminalState>({ status: "connecting" });
  const onState = (next: TerminalState) => {
    if (next.status === "ended" && next.exit === 0) return onClose();
    setState(next);
  };
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="gap-3 sm:max-w-[min(64rem,calc(100%-2rem))]" onEscapeKeyDown={(e) => e.preventDefault()}>
        <DialogHeader className="flex-row items-center gap-3 pr-8">
          <DialogTitle className="min-w-0 truncate font-mono">{title}</DialogTitle>
          <TerminalStatusMarker state={state} />
          <DialogDescription className="sr-only">A shell on {title}. Closing this ends it.</DialogDescription>
        </DialogHeader>
        <TerminalScreen socketPath={socketPath} onState={onState} />
      </DialogContent>
    </Dialog>
  );
};
