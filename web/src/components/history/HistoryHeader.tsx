import { DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";

interface HistoryHeaderProps {
  name: string;
  /** Unknown until the history loads. */
  count: number | undefined;
  showChanges: boolean;
  onShowChanges: (show: boolean) => void;
}

export const HistoryHeader = ({ name, count, showChanges, onShowChanges }: HistoryHeaderProps) => (
  <DialogHeader className="flex-row items-start gap-3 py-4 pr-14 pl-6">
    <div className="min-w-0 flex-1 space-y-0.5">
      <DialogTitle>History</DialogTitle>
      <DialogDescription className="flex gap-1">
        <span className="min-w-0 truncate" title={name}>
          {name}
        </span>
        {count !== undefined && <span className="shrink-0">· {count} {count === 1 ? "version" : "versions"} kept</span>}
      </DialogDescription>
    </div>
    <label className="mt-0.5 flex shrink-0 items-center gap-2 text-sm text-muted-foreground">
      <Switch size="sm" checked={showChanges} onCheckedChange={onShowChanges} />
      Show changes
    </label>
  </DialogHeader>
);
