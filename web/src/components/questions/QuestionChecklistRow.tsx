import type { ReactNode } from "react";
import { CircleCheck, CircleMinus } from "lucide-react";

import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { answerLine, rowProgress, type ChecklistRow, type RowProgress } from "@nexul/client-core/questionChecklist";
import { cn } from "@/lib/utils";

interface QuestionChecklistRowProps {
  row: ChecklistRow;
  number: number;
  open: boolean;
  readOnly: boolean;
  onToggle: () => void;
  // What a skipped row says under its question.
  skippedLabel?: string;
  // The question form, shown under the title while the row is open.
  children: ReactNode;
}

interface RowMarkerProps {
  progress: RowProgress;
  open: boolean;
  number: number;
}

// Drawn on lucide's 24-unit circle so the number and the icons share one ring; y 16.3 centres a mono digit's ink.
const NumberMarker = ({ number, open, flagged }: { number: number; open: boolean; flagged: boolean }) => (
  <svg aria-hidden viewBox="0 0 24 24" className="size-5 shrink-0 overflow-visible font-mono tabular-nums">
    <circle cx="12" cy="12" r="10" strokeWidth="2" className={cn("fill-none stroke-border", open && "fill-foreground stroke-foreground")} />
    <text x="12" y={number > 9 ? 15.9 : 16.3} textAnchor="middle" fontSize={number > 9 ? 10.5 : 12} letterSpacing={number > 9 ? -0.5 : 0} className={open ? "fill-background" : "fill-muted-foreground"}>
      {number}
    </text>
    {flagged && <circle cx="20.5" cy="3.5" r="3.5" className="fill-warning" />}
  </svg>
);

// One glyph per row: a check or a dash once settled, else its number, flagged when a draft or suggestion waits on review.
const RowMarker = ({ progress, open, number }: RowMarkerProps) => {
  if (!open && progress === "answered") return <CircleCheck aria-hidden className="size-5 shrink-0 text-success" />;
  if (!open && progress === "skipped") return <CircleMinus aria-hidden className="size-5 shrink-0 text-muted-foreground" />;
  return <NumberMarker number={number} open={open} flagged={progress === "drafted"} />;
};

const STATUS_LABEL: Record<RowProgress, string> = { answered: "Answered", skipped: "Skipped", drafted: "Needs review", waiting: "Not answered" };

const fromPart = (sources: number | undefined, from: string): string => {
  if (sources !== undefined && sources > 0) return ` from ${sources} ${sources === 1 ? "source" : "sources"}`;
  return from === "" ? "" : ` from ${from}`;
};

// One question as a hairline row: its marker, title, and one plain line of where it stands; the open row holds the question under it.
export const QuestionChecklistRow = ({ row, number, open, readOnly, onToggle, skippedLabel = "Skipped", children }: QuestionChecklistRowProps) => {
  const progress = rowProgress(row);
  const waiting = progress === "drafted" || progress === "waiting";
  const suggested = row.draft?.state === "suggested";
  const drafted = row.draft?.state === "open";
  const from = fromPart(row.draft?.sources, row.draft?.from ?? "");
  const summary = (
    <>
      <RowMarker progress={progress} open={open} number={number} />
      <span className="sr-only">{STATUS_LABEL[progress]}: </span>
      <span className="min-w-0 flex-1">
        <span className={cn("block text-sm text-balance", waiting || open ? "font-medium text-foreground" : "text-muted-foreground")}>
          {row.item.text}
        </span>
        {!open && progress === "answered" && <span className="block truncate text-xs text-muted-foreground">{answerLine(row.answer)}</span>}
        {!open && suggested && <span className="block truncate text-xs text-muted-foreground">Suggested change{from}</span>}
        {!open && drafted && <span className="block truncate text-xs text-muted-foreground">Drafted{from}</span>}
        {!open && progress === "skipped" && <span className="block truncate text-xs text-muted-foreground">{skippedLabel}</span>}
      </span>
    </>
  );

  return (
    <li className="relative border-b border-border">
      {open && <CurrentRowEdge />}
      {readOnly && <div className={cn("flex min-h-11 items-start gap-3 rounded-md px-2.5 py-3", open && currentRowFillClass)}>{summary}</div>}
      {!readOnly && (
        <button
          type="button"
          aria-expanded={open}
          onClick={onToggle}
          className={cn(
            "flex min-h-11 w-full cursor-pointer items-start gap-3 rounded-md px-2.5 py-3 text-left focus-visible:-outline-offset-3!",
            open && currentRowFillClass,
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
