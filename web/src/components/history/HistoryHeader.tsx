import { HistoryViewToggle } from "@/components/history/HistoryViewToggle";
import { DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useMinWidth } from "@/hooks/useMinWidth";
import type { HistoryView } from "@/models/History";

interface HistoryHeaderProps {
  name: string;
  /** The view control, only while the picked version has an older one to compare with; it sits here below 1024px and in the strip above. */
  view?: HistoryView | undefined;
  onView?: ((view: HistoryView) => void) | undefined;
}

export const HistoryHeader = ({ name, view, onView }: HistoryHeaderProps) => {
  const wide = useMinWidth(1024);
  return (
    <div className="flex shrink-0 items-center gap-3 border-b border-border bg-popover py-3 pr-14 pl-6">
      <div className="min-w-0 flex-1">
        <DialogTitle className="type-display text-xl">History</DialogTitle>
        <DialogDescription className="truncate text-row text-muted-foreground" title={name}>
          {name}
        </DialogDescription>
      </div>
      {!wide && view && onView && <HistoryViewToggle view={view} onView={onView} />}
    </div>
  );
};
