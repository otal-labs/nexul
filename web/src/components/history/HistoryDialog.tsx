import { useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryBody } from "@/components/history/HistoryBody";
import { HistoryFooter } from "@/components/history/HistoryFooter";
import { HistoryHeader } from "@/components/history/HistoryHeader";
import { HistoryVersionList } from "@/components/history/HistoryVersionList";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { useFetchHistory } from "@/hooks/HistoryHooks";
import type { HistoryRestore } from "@/models/History";

interface HistoryDialogProps {
  /** The entity's history route, and the route of one version. */
  historyUrl: string;
  versionUrl: (version: number) => string;
  /** The item's name, under the dialog's title. */
  name: string;
  restore: HistoryRestore;
  onClose: () => void;
}

// One dialog for every item that keeps versions; mounted while open, so each open reads the list again.
export const HistoryDialog = ({ historyUrl, versionUrl, name, restore, onClose }: HistoryDialogProps) => {
  const { data: history, error, isPending } = useFetchHistory(historyUrl);
  const [picked, setPicked] = useState<number>();
  const [showChanges, setShowChanges] = useState(true);

  const versions = history?.versions;
  const selected = picked ?? versions?.[0]?.version;
  const index = versions?.findIndex((v) => v.version === selected) ?? -1;
  const current = versions?.[index];
  const previous = versions?.[index + 1];

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="h-[min(46rem,calc(100dvh-4rem))] gap-0 overflow-hidden p-0 sm:max-w-[min(64rem,calc(100%-4rem))]">
        <HistoryHeader name={name} count={versions?.length} showChanges={showChanges} onShowChanges={setShowChanges} />
        {isPending && <LoadingDisplay className="flex-1" />}
        {error && <ErrorDisplay error={error} title="Couldn't load the history." className="m-6" />}
        {versions && versions.length === 0 && <NoDataDisplay message="No versions yet" className="flex-1" />}
        {versions && current && (
          <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_17rem] border-t border-border">
            <HistoryBody versionUrl={versionUrl} version={current.version} previous={previous?.version} showChanges={showChanges} />
            <HistoryVersionList versions={versions} selected={current.version} onSelect={setPicked} />
          </div>
        )}
        {versions && current && <HistoryFooter version={current.version} since={previous?.created_at} isCurrent={current.current} restore={restore} />}
      </DialogContent>
    </Dialog>
  );
};
