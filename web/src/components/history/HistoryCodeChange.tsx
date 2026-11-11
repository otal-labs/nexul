import { HistoryCodeLine } from "@/components/history/HistoryCodeLine";
import type { CodeRun } from "@/utils/HistoryCodeUtility";

const WORDS = { added: "added", removed: "removed", edited: "edited", same: "" } as const;

interface HistoryCodeChangeProps {
  run: CodeRun;
  changes: number;
}

// One hunk of changed lines is one stop for the stepper.
export const HistoryCodeChange = ({ run, changes }: HistoryCodeChangeProps) => (
  <div role="group" aria-label={`Change ${run.change} of ${changes}, ${WORDS[run.mark]}`} data-change={run.change} tabIndex={-1} className="col-span-full grid grid-cols-subgrid">
    {run.lines.map((line, i) => (
      <HistoryCodeLine key={i} line={line} numbered />
    ))}
  </div>
);
