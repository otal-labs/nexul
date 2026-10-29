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
    <div className="flex h-14 shrink-0 items-center gap-3 bg-popover pt-0.5 pr-14 pl-6">
      <DialogTitle className="text-[13px]">History</DialogTitle>
      <span aria-hidden className="h-3 w-px shrink-0 bg-border" />
      <DialogDescription className="min-w-0 flex-1 truncate text-[13px]" title={name}>
        {name}
      </DialogDescription>
      {!wide && view && onView && <HistoryViewToggle view={view} onView={onView} />}
    </div>
  );
};
