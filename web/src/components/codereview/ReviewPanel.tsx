import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { microheaderClass } from "@/components/Microheader";
import { ReviewRow } from "@/components/codereview/ReviewRow";
import { useFetchReviewsByTicket } from "@/hooks/ReviewHooks";
import { cn } from "@/lib/utils";

interface ReviewPanelProps {
  ticketId: string;
}

// Renders nothing with zero reviews — Development's empty line already covers "nothing linked here".
export const ReviewPanel = ({ ticketId }: ReviewPanelProps) => {
  const { data, error } = useFetchReviewsByTicket(ticketId);

  return (
    <>
      {error && <ErrorDisplay error={error} title="Couldn't load reviews." />}
      {data && data.length > 0 && (
        <section className="space-y-3">
          <h2 className={cn(microheaderClass, "px-2 pb-1")}>
            Reviews <span className="tabular-nums">({data.length})</span>
          </h2>
          <EnterList className="divide-y divide-border rounded-md border border-border bg-card">
            {data.map((review) => (
              <ReviewRow key={review.id} review={review} />
            ))}
          </EnterList>
        </section>
      )}
    </>
  );
};
