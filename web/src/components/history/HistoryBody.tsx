import { useMemo } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryDocument } from "@/components/history/HistoryDocument";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchHistoryVersion } from "@/hooks/HistoryHooks";
import { diffWords, type DiffSegment } from "@/utils/DiffUtility";
import { bodyToBlocks, diffBlocks, plainRow } from "@/utils/HistoryBlocksUtility";

interface HistoryBodyProps {
  versionUrl: (version: number) => string;
  version: number;
  /** The version just before it in the list; none on the oldest kept, which shows without marks. */
  previous: number | undefined;
  showChanges: boolean;
}

// Two bodies at a time: the picked version and the one before it. Converting and diffing them is the costly part, so each runs once per pair.
export const HistoryBody = ({ versionUrl, version, previous, showChanges }: HistoryBodyProps) => {
  const after = useFetchHistoryVersion(versionUrl(version));
  const before = useFetchHistoryVersion(previous === undefined ? undefined : versionUrl(previous));
  const compare = showChanges && previous !== undefined;

  const afterBlocks = useMemo(() => after.data && bodyToBlocks(after.data.body), [after.data]);
  const beforeBlocks = useMemo(() => before.data && bodyToBlocks(before.data.body), [before.data]);
  const marked = useMemo(() => afterBlocks && beforeBlocks && diffBlocks(beforeBlocks, afterBlocks), [afterBlocks, beforeBlocks]);
  const plain = useMemo(() => afterBlocks?.map(plainRow), [afterBlocks]);

  const rows = compare ? marked : plain;
  const title: DiffSegment[] =
    compare && before.data && after.data
      ? diffWords(before.data.title, after.data.title)
      : [{ type: "context", text: after.data?.title ?? "" }];
  const isPending = after.isPending || (compare && before.isPending);
  const error = after.error ?? (compare ? before.error : null);

  return (
    <div className="@container min-w-0 overflow-y-auto py-6 pr-6 pl-10 @min-[34rem]:pr-10 @min-[34rem]:pl-14">
      {isPending && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load this version." />}
      {rows && <HistoryDocument title={title} rows={rows} />}
    </div>
  );
};
