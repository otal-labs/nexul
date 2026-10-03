import { Check, Minus } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import type { ProtoQuestion } from "@/components/memory/prototype/InterviewPrototypeData";
import { InterviewPrototypeQuestion } from "@/components/memory/prototype/InterviewPrototypeQuestion";
import { useInterviewPrototypeStore } from "@/components/memory/prototype/InterviewPrototypeStore";
import { answerText } from "@/models/Question";
import { cn } from "@/lib/utils";

type RowStatus = "open" | "answered" | "skipped" | "pending";

interface InterviewPrototypeRowProps {
  question: ProtoQuestion;
  number: number;
  progress: string;
  first: boolean;
}

const RowMarker = ({ status, number }: { status: RowStatus; number: number }) => (
  <span
    aria-hidden
    className={cn(
      "flex size-5 shrink-0 items-center justify-center rounded-full font-mono text-[10px] tabular-nums",
      status === "open" && "bg-foreground text-background",
      status === "answered" && "bg-success text-background",
      status === "skipped" && "bg-muted text-muted-foreground",
      status === "pending" && "border border-border text-muted-foreground",
    )}
  >
    {status === "answered" && <Check className="size-3" strokeWidth={3} />}
    {status === "skipped" && <Minus className="size-3" strokeWidth={3} />}
    {(status === "open" || status === "pending") && number}
  </span>
);

// One question as a hairline row: collapsed to its marker, title, and one-line answer, or open with the question under it.
export const InterviewPrototypeRow = ({ question, number, progress, first }: InterviewPrototypeRowProps) => {
  const { isOpen, answer, skipped, changed, open, epoch } = useInterviewPrototypeStore(
    useShallow((s) => ({
      isOpen: s.openId === question.id,
      answer: s.answers[question.id],
      skipped: s.skipped.includes(question.id),
      changed: s.changed.includes(question.id),
      open: s.open,
      epoch: s.epoch,
    })),
  );
  const status: RowStatus = isOpen ? "open" : answer ? "answered" : skipped ? "skipped" : "pending";

  return (
    <li className="border-b border-border">
      <button
        type="button"
        aria-expanded={isOpen}
        onClick={() => open(question.id)}
        className={cn(
          "flex min-h-11 w-full items-center gap-3 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring",
          !isOpen && "cursor-pointer transition-colors duration-150 ease-standard hover:bg-accent/40",
        )}
      >
        <RowMarker status={status} number={number} />
        <span className="min-w-0 flex-1">
          <span className={cn("block text-sm text-balance", isOpen ? "font-medium text-foreground" : "text-muted-foreground")}>
            {question.text}
          </span>
          {status === "answered" && <span className="block truncate text-xs text-muted-foreground/70">{answerText(answer).replaceAll("\n", " · ")}</span>}
          {status === "skipped" && <span className="block text-xs text-muted-foreground/70">Skipped</span>}
        </span>
        {changed && <span className="shrink-0 font-mono text-[11px] text-muted-foreground">changed</span>}
      </button>
      {isOpen && <InterviewPrototypeQuestion key={`${question.id}:${epoch}`} question={question} progress={progress} first={first} />}
    </li>
  );
};
