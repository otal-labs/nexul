import type { ReactNode } from "react";
import { Check, Minus } from "lucide-react";

import { DRAFTS } from "@/components/memory/prototype/SourcesProtoData";
import { rowState, useSourcesProtoStore, type RowState } from "@/components/memory/prototype/SourcesProtoStore";
import type { AnswerValue, QuestionItem } from "@/models/Question";
import { cn } from "@/lib/utils";

export const valueLine = (value: AnswerValue | undefined): string =>
  value ? [...(value.selected ?? []), value.text ?? ""].filter((p) => p !== "").join(" · ").replaceAll("\n", " · ") : "";

const RowMarker = ({ state, open, number }: { state: RowState; open: boolean; number: number }) => (
  <span
    aria-hidden
    className={cn(
      "flex size-5 shrink-0 items-center justify-center rounded-full font-mono text-[10px] tabular-nums",
      open && "bg-foreground text-background",
      !open && state === "answered" && "bg-success text-background",
      !open && state === "skipped" && "bg-muted text-muted-foreground",
      !open && state === "drafted" && "border border-dashed border-muted-foreground text-foreground",
      !open && state === "pending" && "border border-border text-muted-foreground",
    )}
  >
    {!open && state === "answered" && <Check className="size-3" strokeWidth={3} />}
    {!open && state === "skipped" && <Minus className="size-3" strokeWidth={3} />}
    {(open || state === "pending" || state === "drafted") && number}
  </span>
);

const STATE_LABEL: Record<RowState, string> = { answered: "Answered", skipped: "Skipped", drafted: "Drafted", pending: "Not answered" };

interface SourcesProtoRowProps {
  item: QuestionItem;
  number: number;
  children: ReactNode;
}

// The real question row, plus a dashed marker for a draft and a warning dot for a suggested change.
export const SourcesProtoRow = ({ item, number, children }: SourcesProtoRowProps) => {
  const state = useSourcesProtoStore((s) => rowState(s, item.id));
  const answer = useSourcesProtoStore((s) => s.answers[item.id]);
  const suggestion = useSourcesProtoStore((s) => s.suggestions[item.id]);
  const open = useSourcesProtoStore((s) => s.openId === item.id);
  const setOpen = useSourcesProtoStore((s) => s.open);
  return (
    <li className="border-b border-border">
      <button
        type="button"
        aria-expanded={open}
        onClick={() => setOpen(open ? null : item.id)}
        className={cn(
          "flex min-h-11 w-full cursor-pointer items-center gap-3 rounded-sm px-1.5 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring",
          !open && "transition-colors duration-150 ease-standard hover:bg-accent/40",
        )}
      >
        <RowMarker state={state} open={open} number={number} />
        <span className="sr-only">{STATE_LABEL[state]}: </span>
        <span className="min-w-0 flex-1">
          <span className={cn("block text-sm text-balance", open ? "font-medium text-foreground" : "text-muted-foreground")}>{item.text}</span>
          {!open && state === "answered" && !suggestion && (
            <span className="block truncate text-xs text-muted-foreground/70">{valueLine(answer)}</span>
          )}
          {!open && suggestion && (
            <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
              <span role="img" aria-label="suggested change" className="size-1.5 shrink-0 rounded-full bg-warning" />
              <span className="shrink-0 text-foreground">Suggested change</span>
              <span className="truncate">· from {suggestion.from}</span>
            </span>
          )}
          {!open && state === "drafted" && (
            <span className="block truncate text-xs text-muted-foreground">
              Drafted <span className="text-muted-foreground/70">· from {DRAFTS[item.id]?.from}</span>
            </span>
          )}
          {!open && state === "skipped" && <span className="block text-xs text-muted-foreground/70">Skipped</span>}
        </span>
      </button>
      {open && children}
    </li>
  );
};
