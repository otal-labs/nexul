import { Fragment, useMemo } from "react";

import { HistoryCodeChange } from "@/components/history/HistoryCodeChange";
import { HistoryCodeContext } from "@/components/history/HistoryCodeContext";
import { cn } from "@/lib/utils";
import type { HistoryRun } from "@/utils/HistoryBlocksUtility";
import { codeRuns } from "@/utils/HistoryCodeUtility";

interface HistoryCodeDiffProps {
  runs: HistoryRun[];
  changes: number;
}

// Code reads as a unified diff: both sides' line numbers, a sign per line, and unchanged stretches folded around each change.
export const HistoryCodeDiff = ({ runs, changes }: HistoryCodeDiffProps) => {
  const code = useMemo(() => codeRuns(runs), [runs]);
  const numbered = changes > 0;

  return (
    <div
      className={cn(
        "grid overflow-hidden rounded-md border border-border bg-surface-2 font-mono text-xs leading-5 [font-variant-ligatures:none]",
        numbered ? "grid-cols-[auto_auto_2ch_minmax(0,1fr)]" : "grid-cols-[auto_minmax(0,1fr)]",
      )}
    >
      {code.map((run, i) => (
        <Fragment key={i}>
          {run.mark === "same" && <HistoryCodeContext lines={run.lines} first={i === 0} last={i === code.length - 1} numbered={numbered} />}
          {run.mark !== "same" && <HistoryCodeChange run={run} changes={changes} />}
        </Fragment>
      ))}
    </div>
  );
};
