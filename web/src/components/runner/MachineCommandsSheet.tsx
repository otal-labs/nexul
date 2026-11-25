import { MachineCommands } from "@/components/runner/MachineCommands";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { Machine } from "@/models/Runner";

interface MachineCommandsSheetProps {
  machine: Machine;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// A machine's commands: its switch, a command to run on one of its runners, and what ran there.
export const MachineCommandsSheet = ({ machine, open, onOpenChange }: MachineCommandsSheetProps) => (
  <Sheet open={open} onOpenChange={onOpenChange}>
    <SheetContent className="w-full gap-4 overflow-y-auto sm:max-w-xl">
      <SheetHeader>
        <SheetTitle>Commands on {machine.name}</SheetTitle>
        <SheetDescription>Each runs on the runner you pick, as the user it runs as: root on Linux.</SheetDescription>
      </SheetHeader>
      <div className="px-4 pb-4">{open && <MachineCommands machine={machine} />}</div>
    </SheetContent>
  </Sheet>
);
