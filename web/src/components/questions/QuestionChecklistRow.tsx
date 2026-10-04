import type { ReactNode } from "react";
import { Check, Minus } from "lucide-react";

import { answerLine, rowStatus, type ChecklistRow, type ChecklistRowStatus } from "@/models/QuestionChecklist";
import { cn } from "@/lib/utils";

interface QuestionChecklistRowProps {
  row: ChecklistRow;
  number: number;
  open: boolean;
  readOnly: boolean;
  onToggle: () => void;
  // The question form, shown under the title while the row is open.
  children: ReactNode;
}

interface RowMarkerProps {
  status: ChecklistRowStatus;
  drafted: boolean;
  open: boolean;
  number: number;
}

const RowMarker = ({ status, drafted, open, number }: RowMarkerProps) => (
  <span
    aria-hidden
    className={cn(
      "flex size-5 shrink-0 items-center justify-center rounded-full font-mono text-[10px] tabular-nums",
      open && "bg-foreground text-background",
      !open && status === "answered" && "bg-success text-background",
      !open && drafted && "border border-dashed border-muted-foreground text-foreground",
      !open && !drafted && status === "skipped" && "bg-muted text-muted-foreground",
      !open && !drafted && status === "pending" && "border border-border text-muted-foreground",
    )}
  >
    {!open && status === "answered" && <Check className="size-3" strokeWidth={3} />}
    {!open && !drafted && status === "skipped" && <Minus className="size-3" strokeWidth={3} />}
    {(open || drafted || status === "pending") && number}
  </span>
);

const STATUS_LABEL: Record<ChecklistRowStatus, string> = { answered: "Answered", skipped: "Skipped", pending: "Not answered" };

const fromPart = (from: string) => (from === "" ? "" : ` · from ${from}`);

// One question as a hairline row: its marker, title, and one-line answer; the open row holds the question under it.
export const QuestionChecklistRow = ({ row, number, open, readOnly, onToggle, children }: QuestionChecklistRowProps) => {
  const status = rowStatus(row);
  const drafted = row.draft?.state === "open";
  const suggested = row.draft?.state === "suggested";
  const from = fromPart(row.draft?.from ?? "");
  const summary = (
    <>
      <RowMarker status={status} drafted={drafted} open={open} number={number} />
      <span className="sr-only">{drafted ? "Drafted" : STATUS_LABEL[status]}: </span>
      <span className="min-w-0 flex-1">
        <span className={cn("block text-sm text-balance", open ? "font-medium text-foreground" : "text-muted-foreground")}>
          {row.item.text}
        </span>
        {!open && status === "answered" && !suggested && (
          <span className="block truncate text-xs text-muted-foreground/70">{answerLine(row.answer)}</span>
        )}
        {!open && suggested && (
          <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
            <span aria-hidden className="size-1.5 shrink-0 rounded-full bg-warning" />
            <span className="shrink-0 text-foreground">Suggested change</span>
            <span className="truncate">{from}</span>
          </span>
        )}
        {!open && drafted && (
          <span className="block truncate text-xs text-muted-foreground">
            Drafted<span className="text-muted-foreground/70">{from}</span>
          </span>
        )}
        {!open && !drafted && status === "skipped" && <span className="block text-xs text-muted-foreground/70">Skipped</span>}
      </span>
    </>
  );

  return (
    <li className="border-b border-border">
      {readOnly && <div className="flex min-h-11 items-center gap-3 px-1.5 py-2.5">{summary}</div>}
      {!readOnly && (
        <button
          type="button"
          aria-expanded={open}
          onClick={onToggle}
          className={cn(
            "flex min-h-11 w-full cursor-pointer items-center gap-3 rounded-sm px-1.5 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring",
            !open && "transition-colors duration-150 ease-standard hover:bg-accent/40",
          )}
        >
          {summary}
        </button>
      )}
      {open && children}
    </li>
  );
};
