import { progressCounts, rowProgress, type RowProgress } from "@nexul/client-core/questionChecklist";
import type { InterviewSectionData } from "@/models/InterviewAnswer";
import { cn } from "@/lib/utils";

interface InterviewProgressProps {
  sections: InterviewSectionData[];
  current: string | null;
  // Opens the first question that needs the person, unfolding its round.
  onJump: () => void;
}

const SEGMENT: Record<RowProgress, string> = {
  answered: "bg-foreground/55",
  skipped: "bg-foreground/20 ring-1 ring-foreground/45 ring-inset",
  drafted: "bg-warning-mark/45",
  waiting: "bg-foreground/10",
};

const LEGEND: { kind: RowProgress; label: string }[] = [
  { kind: "answered", label: "Answered" },
  { kind: "skipped", label: "Skipped" },
  { kind: "drafted", label: "Drafted" },
  { kind: "waiting", label: "Unanswered" },
];

// The whole interview as one meter, a segment per question grouped by round, over the one count line it is drawn from.
export const InterviewProgress = ({ sections, current, onJump }: InterviewProgressProps) => {
  const { answered, skipped, needsYou, total } = progressCounts(sections.flatMap((s) => s.rows));
  const line = [`${answered} of ${total} answered`, skipped > 0 && `${skipped} skipped`].filter(Boolean).join(" · ");
  const need = `${needsYou} need${needsYou === 1 ? "s" : ""} you`;
  const shown = new Set(sections.flatMap((s) => s.rows.filter((r) => r.key !== current).map(rowProgress)));
  return (
    <div className="space-y-2">
      <div role="img" aria-label={needsYou > 0 ? `${line} · ${need}` : line} className="flex h-1 gap-2.5">
        {sections.map((section) => (
          <div key={section.key} className="flex gap-0.5" style={{ flexGrow: section.rows.length, flexBasis: 0 }}>
            {section.rows.map((row) => (
              <span key={row.key} className={cn("h-1 flex-1 rounded-full", row.key === current ? "bg-brand" : SEGMENT[rowProgress(row)])} />
            ))}
          </div>
        ))}
      </div>
      <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
        <p className="flex flex-wrap items-baseline gap-x-1.5 font-mono text-xs text-muted-foreground tabular-nums">
          <span>{line}</span>
          {needsYou > 0 && (
            <>
              <span aria-hidden>·</span>
              <button type="button" onClick={onJump} className="cursor-pointer rounded-sm text-foreground underline-offset-2 hover:underline">
                {need}
              </button>
            </>
          )}
        </p>
        <ul aria-hidden className="ml-auto flex flex-wrap items-center gap-x-3 text-[11px] text-muted-foreground">
          {current !== null && <LegendItem swatch="bg-brand" label="Open" />}
          {LEGEND.filter((l) => shown.has(l.kind)).map((l) => <LegendItem key={l.kind} swatch={SEGMENT[l.kind]} label={l.label} />)}
        </ul>
      </div>
    </div>
  );
};

const LegendItem = ({ swatch, label }: { swatch: string; label: string }) => (
  <li className="flex items-center gap-1.5">
    <span className={cn("h-1 w-3 rounded-full", swatch)} />
    {label}
  </li>
);
