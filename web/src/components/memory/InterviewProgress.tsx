import { progressCounts } from "@nexul/client-core/questionChecklist";
import type { InterviewSectionData } from "@/models/InterviewAnswer";

interface InterviewProgressProps {
  sections: InterviewSectionData[];
  // Opens the first question that needs the person, unfolding its round.
  onJump: () => void;
}

// The whole interview as one meter over the one count line it is drawn from.
export const InterviewProgress = ({ sections, onJump }: InterviewProgressProps) => {
  const { answered, skipped, needsYou, total } = progressCounts(sections.flatMap((s) => s.rows));
  const line = [`${answered} of ${total} answered`, skipped > 0 && `${skipped} skipped`].filter(Boolean).join(" · ");
  const need = `${needsYou} need${needsYou === 1 ? "s" : ""} you`;
  return (
    <div className="space-y-2">
      <div role="img" aria-label={needsYou > 0 ? `${line} · ${need}` : line} className="h-1 overflow-hidden rounded-full bg-foreground/10">
        <div className="h-full rounded-full bg-brand" style={{ width: `${total === 0 ? 0 : (answered / total) * 100}%` }} />
      </div>
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
    </div>
  );
};
