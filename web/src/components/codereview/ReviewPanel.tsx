import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ReviewRow } from "@/components/codereview/ReviewRow";
import { useFetchReviewsByTicket } from "@/hooks/ReviewHooks";

interface ReviewPanelProps {
  ticketId: string;
}

// Renders nothing with zero reviews — Development's empty line already covers "nothing linked here".
export const ReviewPanel = ({ ticketId }: ReviewPanelProps) => {
  const { data, error } = useFetchReviewsByTicket(ticketId);

  return (
    <>
      {error && <ErrorDisplay error={error} title="Failed to load reviews" />}
      {data && data.length > 0 && (
        <section className="animate-in fade-in-0 slide-in-from-bottom-1 space-y-3 duration-200 ease-out">
          <h2 className="px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
            Reviews <span className="tabular-nums">({data.length})</span>
          </h2>
          <ul className="divide-y divide-border rounded-md border border-border bg-card">
            {data.map((review, index) => (
              <ReviewRow key={review.id} review={review} index={index} />
            ))}
          </ul>
        </section>
      )}
    </>
  );
};
