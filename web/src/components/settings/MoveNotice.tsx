import { Button } from "@/components/ui/button";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { Computer } from "@/models/Pairing";
import { cn } from "@/lib/utils";

const detailOf = (computer: Computer) => {
  if (computer.pair_error) return `Not moved yet: ${computer.pair_error}`;
  if (computer.runner) return "Moving: waiting for Nexul to pair through the app";
  return "Pairing through a tunnel or by URL was retired, so agent work can't run here until it moves. One command moves it, keeping its projects, setup and history.";
};

interface MoveNoticeProps {
  computer: Computer;
  onMove: () => void;
  className?: string;
}

// The notice on a computer paired before the Nexul app: one command moves it, and a move under way says how far it got.
export const MoveNotice = ({ computer, onMove, className }: MoveNoticeProps) => (
  <div className={cn("mt-2 flex min-w-0 items-center gap-3", className)}>
    <div className="min-w-0 flex-1 space-y-0.5">
      <SettingsStatus tone={computer.pair_error ? "destructive" : "info"}>Move this computer to the Nexul app</SettingsStatus>
      <p className="pl-3 text-xs break-words text-muted-foreground">{detailOf(computer)}</p>
    </div>
    <Button type="button" variant="outline" size="sm" onClick={onMove}>
      {computer.runner ? "Show progress" : "Move"}
    </Button>
  </div>
);
