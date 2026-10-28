import { microheaderClass } from "@/components/Microheader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { AutoPlaySignals } from "@/components/play/AutoPlaySignals";
import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { PlayRailRow } from "@/components/play/PlayRailRow";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import { useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { latestRunOf } from "@/models/Trail";
import type { Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";
import { hasAutoPlaySignals } from "@/utils/PlayQueueUtility";

interface PlaysRailSectionProps {
  ticket: Ticket;
}

// Plays are stage-bound, so the section appears when a play applies, the load failed, or an auto play waits or is paused.
export const PlaysRailSection = ({ ticket }: PlaysRailSectionProps) => {
  const { data: applicable = [], error } = useApplicableTicketPlays(ticket);
  const { data: queue } = useFetchPlayQueue("ticket", ticket.id);
  // Newest first, the same list the Trail section shows.
  const { data: trails } = useFetchTrails("ticket", ticket.id);
  const plays = applicable.filter((p) => p.can_run);
  const hasPlays = plays.length > 0;
  const failed = Boolean(error);
  const signals = hasAutoPlaySignals(queue);

  return (
    (hasPlays || failed || signals) && (
      <section className="space-y-0.5">
        <h2 className={cn(microheaderClass, "px-2 pb-1")}>Plays</h2>
        {failed && <ErrorDisplay error={error} title="Couldn't load plays." />}
        {hasPlays && <HarnessReadinessNote projectId={ticket.project_id} className="px-2 pb-1" />}
        {signals && <AutoPlaySignals targetType="ticket" targetId={ticket.id} developer={ticket.developer_id} />}
        {hasPlays && (
          <div className="flex flex-col">
            {plays.map((play) => (
              <PlayRailRow key={play.id} play={play} ticket={ticket} lastRun={latestRunOf(trails, play.id)} />
            ))}
          </div>
        )}
      </section>
    )
  );
};
