import type { DiffSegment } from "@/utils/DiffUtility";
import type { HistoryRow, HistoryRun, RowMark } from "@/utils/HistoryBlocksUtility";

export type CodeSign = "context" | "add" | "remove";

export interface CodeLine {
  sign: CodeSign;
  /** The line's number in the older version and in the newer one; a removed line has no new number, an added one no old. */
  oldNo?: number;
  newNo?: number;
  /** An edited line's changed words are its add or remove parts; a whole added or removed line is one context part. */
  parts: DiffSegment[];
}

export interface CodeRun {
  mark: RowMark;
  change: number;
  lines: CodeLine[];
}

// Unchanged lines a diff keeps on each side of a change, as git does.
export const CODE_CONTEXT = 3;
// The fewest unchanged lines worth folding behind a row of their own.
const MIN_FOLD = 4;

const whole = (row: HistoryRow): DiffSegment[] => [{ type: "context", text: row.block.text }];

// Runs as git reads them: a changed run lists its old lines, then its new ones, an edited line on both sides with its own words marked.
export const codeRuns = (runs: HistoryRun[]): CodeRun[] => {
  let oldNo = 0;
  let newNo = 0;
  return runs.map((run) => {
    if (run.mark === "same") {
      return { mark: run.mark, change: run.change, lines: run.rows.map((row) => ({ sign: "context", oldNo: ++oldNo, newNo: ++newNo, parts: whole(row) })) };
    }
    const removed = run.rows
      .filter((row) => row.mark !== "added")
      .map((row): CodeLine => ({ sign: "remove", oldNo: ++oldNo, parts: row.mark === "edited" ? row.parts.filter((p) => p.type !== "add") : whole(row) }));
    const added = run.rows
      .filter((row) => row.mark !== "removed")
      .map((row): CodeLine => ({ sign: "add", newNo: ++newNo, parts: row.mark === "edited" ? row.parts.filter((p) => p.type !== "remove") : whole(row) }));
    return { mark: run.mark, change: run.change, lines: [...removed, ...added] };
  });
};

export interface CodeFold {
  head: CodeLine[];
  hidden: CodeLine[];
  tail: CodeLine[];
}

// An unchanged run between changes keeps three lines by each change and folds the rest; with no change at all nothing folds.
export const foldContext = (lines: CodeLine[], first: boolean, last: boolean, hasChanges: boolean): CodeFold => {
  const head = first ? 0 : CODE_CONTEXT;
  const tail = last ? 0 : CODE_CONTEXT;
  if (!hasChanges || lines.length - head - tail < MIN_FOLD) return { head: lines, hidden: [], tail: [] };
  return { head: lines.slice(0, head), hidden: lines.slice(head, lines.length - tail), tail: lines.slice(lines.length - tail) };
};
