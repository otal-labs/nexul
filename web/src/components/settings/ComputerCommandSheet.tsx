import { Suspense } from "react";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { LogsView } from "@/components/logs/LogsView";
import { LazyTerminalTranscript } from "@/components/terminal/LazyTerminal";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { commandOutputSource, type ComputerCommand } from "@/models/ComputerCommand";

interface ComputerCommandSheetProps {
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
export const ComputerCommandSheet = ({ command, open, onOpenChange }: ComputerCommandSheetProps) => (
  <Sheet open={open} onOpenChange={onOpenChange}>
    <SheetContent className="w-full gap-4 overflow-y-auto sm:max-w-3xl">
      <SheetHeader>
        <SheetTitle className="font-mono break-all">{command.command}</SheetTitle>
        <SheetDescription>{outputNote(command) || (command.adapter === "terminal" ? "A terminal" : `In ${command.cwd || "the home folder"}`)}</SheetDescription>
      </SheetHeader>
      <div className="px-4 pb-4">
        {command.adapter === "terminal" && (
          <Suspense fallback={<LoadingDisplay label="Loading the terminal" />}>
            <LazyTerminalTranscript command={command} />
          </Suspense>
        )}
        {command.adapter !== "terminal" && <LogsView key={command.id} source={commandOutputSource(command)} />}
      </div>
    </SheetContent>
  </Sheet>
);
