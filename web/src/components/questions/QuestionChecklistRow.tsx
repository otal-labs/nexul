import type { ReactNode } from "react";
import { CircleCheck, CircleMinus } from "lucide-react";

import { answerLine, rowStatus, type ChecklistRow, type ChecklistRowStatus } from "@nexul/client-core/questionChecklist";
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

// Drawn on lucide's 24-unit circle so the number and the icons share one ring; y 16.3 centres a mono digit's ink.
const NumberMarker = ({ number, open, drafted }: { number: number; open: boolean; drafted: boolean }) => (
  <svg aria-hidden viewBox="0 0 24 24" className="size-5 shrink-0 font-mono tabular-nums">
    <circle
      cx="12"
      cy="12"
      r="10"
      strokeWidth="2"
      strokeDasharray={drafted && !open ? "3.5 2.5" : undefined}
      className={cn("fill-none stroke-border", drafted && "stroke-muted-foreground", open && "fill-foreground stroke-foreground")}
    />
    <text x="12" y="16.3" textAnchor="middle" fontSize="12" className={cn("fill-muted-foreground", drafted && "fill-foreground", open && "fill-background")}>
      {number}
    </text>
  </svg>
);

const RowMarker = ({ status, drafted, open, number }: RowMarkerProps) => {
  if (open || drafted || status === "pending") return <NumberMarker number={number} open={open} drafted={drafted} />;
  if (status === "answered") return <CircleCheck aria-hidden className="size-5 shrink-0 text-success" />;
  return <CircleMinus aria-hidden className="size-5 shrink-0 text-muted-foreground" />;
};

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
      {readOnly && <div className="flex min-h-11 items-start gap-3 px-1.5 py-3">{summary}</div>}
      {!readOnly && (
        <button
          type="button"
          aria-expanded={open}
          onClick={onToggle}
          className={cn(
            "flex min-h-11 w-full cursor-pointer items-start gap-3 rounded-sm px-1.5 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring",
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
