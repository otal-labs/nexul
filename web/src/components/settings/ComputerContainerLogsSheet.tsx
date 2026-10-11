import { LogsView } from "@/components/logs/LogsView";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { computerLogSource } from "@/models/ContainerLog";

interface ComputerContainerLogsSheetProps {
  computerId: string;
  container: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// The stack page's log viewer over one container on the computer; closing the sheet stops docker logs there.
export const ComputerContainerLogsSheet = ({ computerId, container, open, onOpenChange }: ComputerContainerLogsSheetProps) => (
  <Sheet open={open} onOpenChange={onOpenChange}>
    <SheetContent className="w-full gap-4 overflow-y-auto sm:max-w-3xl">
      <SheetHeader>
        <SheetTitle className="font-mono break-all">{container}</SheetTitle>
        <SheetDescription>Read live from Docker on this computer. Nexul stores none of it.</SheetDescription>
      </SheetHeader>
      <div className="px-4 pb-4">
        <LogsView key={container} source={computerLogSource(computerId, container)} />
      </div>
    </SheetContent>
  </Sheet>
);
