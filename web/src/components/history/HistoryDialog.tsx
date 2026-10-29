import { useRef, useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryCompare } from "@/components/history/HistoryCompare";
import { HistoryFooter } from "@/components/history/HistoryFooter";
import { HistoryHeader } from "@/components/history/HistoryHeader";
import { HistoryOneVersion } from "@/components/history/HistoryOneVersion";
import { HistoryVersionList } from "@/components/history/HistoryVersionList";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useHistoryCompare } from "@/hooks/HistoryCompareHooks";
import { useFetchHistory } from "@/hooks/HistoryHooks";
import { useChangeStepper } from "@/hooks/HistoryStepperHooks";
import { cn } from "@/lib/utils";
import { restoredLabel } from "@/utils/HistoryVersionUtility";
import type { HistoryRestore, HistoryView } from "@/models/History";

interface HistoryDialogProps {
  /** The entity's history route, and the route of one version. */
  historyUrl: string;
  versionUrl: (version: number) => string;
  /** The item's name, in the header. */
  name: string;
  restore: HistoryRestore;
  onClose: () => void;
}

// One dialog for every item that keeps versions; mounted while open, so each open reads the list again.
export const HistoryDialog = ({ historyUrl, versionUrl, name, restore, onClose }: HistoryDialogProps) => {
  const { data: history, error, isPending } = useFetchHistory(historyUrl);
  const [picked, setPicked] = useState<number>();
  const [view, setView] = useState<HistoryView>("changes");
  const scroller = useRef<HTMLDivElement>(null);

  const versions = history?.versions;
  const selected = picked ?? versions?.[0]?.version;
  const index = versions?.findIndex((v) => v.version === selected) ?? -1;
  const current = versions?.[index];
  const previous = versions?.[index + 1];

  const compare = useHistoryCompare(versionUrl, current?.version, previous?.version, view === "changes");
  const stepper = useChangeStepper(scroller, compare);
  const single = versions?.length === 1;

  const pick = (version: number) => {
    setPicked(version);
    stepper.reset();
  };
  const changeView = (next: HistoryView) => {
    setView(next);
    stepper.reset();
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        onKeyDown={stepper.onKeyDown}
        className={cn("gap-0 overflow-hidden p-0", single ? "sm:max-w-[30rem]" : "h-[min(46rem,calc(100dvh-4rem))] sm:max-w-[min(64rem,calc(100%-4rem))]")}
      >
        <HistoryHeader name={name} view={previous && view} onView={previous && changeView} />
        {isPending && <LoadingDisplay className="flex-1" />}
        {error && <ErrorDisplay error={error} title="Couldn't load the history." className="m-6" />}
        {versions && versions.length === 0 && <NoDataDisplay message="No versions yet" className="flex-1" />}
        {versions && current && single && <HistoryOneVersion version={current} />}
        {versions && current && !single && (
          <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_auto] border-t border-border">
            <HistoryCompare ref={scroller} picked={current} restored={restoredLabel(current, versions)} previous={previous} compare={compare} view={view} onView={changeView} step={stepper.step} onHop={stepper.hop} />
            <HistoryVersionList versions={versions} changes={history.changes} selected={current.version} onSelect={pick} />
          </div>
        )}
        {versions && current && !single && <HistoryFooter version={current.version} savedAt={current.created_at} isCurrent={current.current} restore={restore} />}
      </DialogContent>
    </Dialog>
  );
};
