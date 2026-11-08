import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { SweepLineRow } from "@/components/sweep/SweepLineRow";
import { useFetchSweep } from "@/hooks/SweepHooks";
import { countsPhrase, inSentence, type SweepDigest } from "@/models/Schedule";
import { formatCalendarTime } from "@/utils/TimeUtility";

// "Ended today at 04:12: 4 finished, 1 asked. 3 more matched past its cap."
const summaryOf = (d: SweepDigest) => {
  const when = d.finished_at ? `Ended ${inSentence(formatCalendarTime(d.finished_at))}` : "Still going";
  const counts = countsPhrase(d.counts);
  const over = d.over_cap > 0 ? ` ${d.over_cap} more matched past its cap.` : "";
  return `${when}${counts ? `: ${counts}.` : "."}${over}`;
};

// One sweep: when it fired, how it went, and a row per ticket.
export const SweepDigestView = ({ sweepId }: { sweepId: string }) => {
  const { data, error, isPending } = useFetchSweep(sweepId);
  return (
    <div className="space-y-3">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <header className="space-y-1">
          <h2 className="text-base font-medium text-balance">
            {data.play_label} swept {inSentence(formatCalendarTime(data.fired_at))}
          </h2>
          <p className="text-sm text-muted-foreground">{summaryOf(data)}</p>
        </header>
      )}
      {data && data.lines.length === 0 && <NoDataDisplay message="No ticket matched." />}
      {data && data.lines.length > 0 && (
        <ul>
          {data.lines.map((line) => (
            <SweepLineRow key={line.queue_item_id} line={line} />
          ))}
        </ul>
      )}
    </div>
  );
};
