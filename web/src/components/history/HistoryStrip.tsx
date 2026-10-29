import { HistoryStepper } from "@/components/history/HistoryStepper";
import { HistoryViewToggle } from "@/components/history/HistoryViewToggle";
import { useMinWidth } from "@/hooks/useMinWidth";
import type { HistoryVersionSummary, HistoryView } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

const monoClass = "font-mono text-xs tabular-nums whitespace-nowrap";

// The word "Restored" at every width, with where it came from in its tooltip.
const RestoredTag = ({ label }: { label: string }) => (
  <span title={label} className="shrink-0 rounded-sm border border-border px-1.5 py-0.5 text-[11px] leading-none text-muted-foreground">
    Restored
  </span>
);

interface HistoryStripProps {
  picked: HistoryVersionSummary;
  /** "Restored from 24 Oct 16:24" when the picked version is a restore, else empty. */
  restored: string;
  /** The version before it; none on the oldest kept. */
  previous: HistoryVersionSummary | undefined;
  view: HistoryView;
  onView: (view: HistoryView) => void;
  /** False until both texts are read, so the strip never says "No text changes" while it is still loading. */
  ready: boolean;
  changes: number;
  step: number;
  onHop: (to: number) => void;
}

// Which two versions are being compared and where in their changes the reader is; the one place the diff is steered from. The
// left slot holds the count, or what stands in for it, and the picked time always follows it.
export const HistoryStrip = ({ picked, restored, previous, view, onView, ready, changes, step, onHop }: HistoryStripProps) => {
  const wide = useMinWidth(1024);
  const comparing = !!previous && view === "changes";
  const time = formatVersionTime(picked.created_at);

  return (
    <div className="flex h-11 shrink-0 items-center gap-3 border-b border-border bg-popover px-10 @min-[34rem]:px-14">
      {comparing && ready && <HistoryStepper changes={changes} step={step} onHop={onHop} wide={wide} />}
      {view === "changes" && !previous && <span className="shrink-0 text-[13px] font-medium whitespace-nowrap">Oldest kept version</span>}
      {view === "changes" && (!previous || ready) && <span aria-hidden className="h-3 w-px shrink-0 bg-border" />}
      <span className="flex min-w-0 flex-1 items-baseline gap-2">
        {view === "as-it-was" && (
          <span className="truncate text-[13px] text-muted-foreground">
            Showing <span className={`${monoClass} text-foreground`}>{time}</span> as it was
          </span>
        )}
        {view === "changes" && <span className={monoClass}>{time}</span>}
        {view === "changes" && restored && <RestoredTag label={restored} />}
        {comparing && previous && (wide || !restored) && <span className={`${monoClass} min-w-0 truncate text-muted-foreground`}>since {formatVersionTime(previous.created_at)}</span>}
      </span>
      {wide && previous && <HistoryViewToggle view={view} onView={onView} />}
    </div>
  );
};
