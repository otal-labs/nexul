import { editedBarClass, HistoryRunBox } from "@/components/history/HistoryRunBox";
import { HistoryText } from "@/components/history/HistoryText";
import { displayTitleClass } from "@/components/PageHeader";
import { cn } from "@/lib/utils";
import type { DiffSegment } from "@/utils/DiffUtility";
import type { HistoryRun } from "@/utils/HistoryBlocksUtility";

interface HistoryDocumentProps {
  title: DiffSegment[];
  /** A changed title is change 1: it carries the same margin bar as an edited run, and the stepper stops on it. */
  titleChanged: boolean;
  runs: HistoryRun[];
  changes: number;
}

export const HistoryDocument = ({ title, titleChanged, runs, changes }: HistoryDocumentProps) => (
  <>
    <div
      {...(titleChanged && { role: "group", "aria-label": `Change 1 of ${changes}, edited`, "data-change": 1, tabIndex: -1 })}
      className={cn("mb-4", titleChanged && editedBarClass)}
    >
      <h3 className={`${displayTitleClass} text-2xl`}>
        <HistoryText parts={title} />
      </h3>
    </div>
    <div className="prose-rich text-sm">
      {runs.map((run, i) => (
        <HistoryRunBox key={i} run={run} changes={changes} />
      ))}
    </div>
  </>
);
