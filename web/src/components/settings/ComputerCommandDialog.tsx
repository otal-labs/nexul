import { Suspense } from "react";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LogsView } from "@/components/logs/LogsView";
import { LazyTerminalTranscript } from "@/components/terminal/LazyTerminal";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { commandOutputSource, type ComputerCommand } from "@/models/ComputerCommand";

interface ComputerCommandDialogProps {
  command: ComputerCommand;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const outputNote = (c: ComputerCommand) => {
  if (c.output_purged) return "Its output was deleted 30 days after it ended.";
  if (c.output_truncated) return "Only its last MiB of output was kept.";
  return c.error;
};

// One command's output from its first line, live while it runs, in the stack page's log viewer; a terminal's replays in
// its own screen.
export const ComputerCommandDialog = ({ command, open, onOpenChange }: ComputerCommandDialogProps) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="gap-3 sm:max-w-[min(64rem,calc(100%-2rem))]">
      <DialogHeader className="pr-8">
        <DialogTitle className="font-mono break-all">{command.command}</DialogTitle>
        <DialogDescription>{outputNote(command) || (command.adapter === "terminal" ? "A terminal" : `In ${command.cwd || "the home folder"}`)}</DialogDescription>
      </DialogHeader>
      {command.adapter === "terminal" && (
        <Suspense fallback={<LoadingDisplay label="Loading the terminal" />}>
          <LazyTerminalTranscript command={command} />
        </Suspense>
      )}
      {command.adapter !== "terminal" && <LogsView key={command.id} source={commandOutputSource(command)} />}
    </DialogContent>
  </Dialog>
);
