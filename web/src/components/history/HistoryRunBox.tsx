import { HistoryBlockRow } from "@/components/history/HistoryBlockRow";
import { cn } from "@/lib/utils";
import type { HistoryRun, RowMark } from "@/utils/HistoryBlocksUtility";

// Every change sits on one vertical line: an added or removed run is a band that grows left into the margin, so its bar is its
// own left edge; an edited run keeps a muted bar at the same x.
const BAND: Partial<Record<RowMark, string>> = {
  added: "border-success bg-success/10 dark:bg-success/16",
  removed: "border-destructive bg-destructive/10 dark:bg-destructive/16",
};
const BAND_SHAPE = "-mr-2 -ml-5 rounded-r-sm border-l-[3px] py-1.5 pr-2 pl-[17px] [&>div:first-of-type>*]:mt-0! [&>div:last-of-type>*]:mb-0!";
export const editedBarClass = "relative before:absolute before:inset-y-1 before:-left-5 before:w-[3px] before:rounded-sm before:bg-muted-foreground";
const WORDS: Partial<Record<RowMark, string>> = { added: "added", removed: "removed", edited: "edited" };

interface HistoryRunBoxProps {
  run: HistoryRun;
  /** How many changes the version holds, for the run's name. */
  changes: number;
}

// A run of adjacent changed blocks is one change: one unbroken bar, one stop for the stepper.
export const HistoryRunBox = ({ run, changes }: HistoryRunBoxProps) => {
  const blocks = run.rows.map((row, i) => <HistoryBlockRow key={i} row={row} />);
  return (
    <>
      {run.mark === "same" && blocks}
      {run.mark !== "same" && (
        <div
          role="group"
          aria-label={`Change ${run.change} of ${changes}, ${WORDS[run.mark]}`}
          data-change={run.change}
          tabIndex={-1}
          className={cn(run.mark === "edited" ? editedBarClass : cn(BAND_SHAPE, BAND[run.mark]))}
        >
          {blocks}
        </div>
      )}
    </>
  );
};
