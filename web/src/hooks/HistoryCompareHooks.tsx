import { useMemo } from "react";

import { useFetchHistoryVersion } from "@/hooks/HistoryHooks";
import type { HistoryBodyKind } from "@/models/History";
import { diffWords, type DiffSegment } from "@/utils/DiffUtility";
import { bodyToBlocks, diffBlocks, groupRuns, plainRow, type HistoryRun } from "@/utils/HistoryBlocksUtility";

export interface HistoryCompare {
  runs: HistoryRun[] | undefined;
  title: DiffSegment[];
  /** A changed title is change 1, ahead of the body's runs. */
  titleChanged: boolean;
  /** How many changes the body shows, the title's included; the stepper's count. */
  changes: number;
  isPending: boolean;
  error: Error | null;
}

// Two bodies at a time: the picked version and the one before it. Converting and diffing them is the costly part, so each runs once per pair.
export const useHistoryCompare = (
  versionUrl: (version: number) => string,
  version: number | undefined,
  /** The version just before it in the list; none on the oldest kept, which shows without marks. */
  previous: number | undefined,
  showChanges: boolean,
  bodyKind: HistoryBodyKind,
): HistoryCompare => {
  const after = useFetchHistoryVersion(version === undefined ? undefined : versionUrl(version));
  const before = useFetchHistoryVersion(previous === undefined ? undefined : versionUrl(previous));
  const compare = showChanges && previous !== undefined;

  const title = useMemo<DiffSegment[]>(() => {
    if (after.data?.title === undefined) return [];
    if (compare && before.data?.title !== undefined) return diffWords(before.data.title, after.data.title);
    return [{ type: "context", text: after.data.title }];
  }, [compare, before.data, after.data]);
  const titleChanged = title.some((part) => part.type !== "context");

  const afterBlocks = useMemo(() => after.data && bodyToBlocks(after.data.body, bodyKind), [after.data, bodyKind]);
  const beforeBlocks = useMemo(() => before.data && bodyToBlocks(before.data.body, bodyKind), [before.data, bodyKind]);
  const runs = useMemo(() => {
    if (!afterBlocks) return undefined;
    if (!compare) return groupRuns(afterBlocks.map(plainRow));
    return beforeBlocks && groupRuns(diffBlocks(beforeBlocks, afterBlocks), titleChanged ? 2 : 1);
  }, [afterBlocks, beforeBlocks, compare, titleChanged]);

  return {
    runs,
    title,
    titleChanged,
    changes: compare ? (runs?.filter((run) => run.change > 0).length ?? 0) + (titleChanged ? 1 : 0) : 0,
    isPending: after.isPending || (compare && before.isPending),
    error: after.error ?? (compare ? before.error : null),
  };
};
