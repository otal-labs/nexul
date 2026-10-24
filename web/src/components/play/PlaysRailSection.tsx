import { microheaderClass } from "@/components/Microheader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { AutoPlaySignals } from "@/components/play/AutoPlaySignals";
import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { PlayButton } from "@/components/play/PlayButton";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import { useFetchPlayQueue } from "@/hooks/PlayQueueHooks";
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
        {signals && <AutoPlaySignals targetType="ticket" targetId={ticket.id} developer={ticket.developer} />}
        {hasPlays && (
          <div className="flex flex-col">
            {plays.map((play) => (
              <PlayButton
                key={play.id}
                play={play}
                projectId={ticket.project_id}
                targetType="ticket"
                targetId={ticket.id}
                variant="ghost"
                className="w-full"
                mainClassName="h-auto min-w-0 flex-1 items-start justify-start gap-2 px-2! py-1.5 text-left text-xs font-normal whitespace-normal [&>svg]:mt-px"
              />
            ))}
          </div>
        )}
      </section>
    )
  );
};
