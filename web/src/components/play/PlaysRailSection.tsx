import { microheaderClass } from "@/components/Microheader";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { PlayButton } from "@/components/play/PlayButton";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";

interface PlaysRailSectionProps {
  ticket: Ticket;
}

// Plays are stage-bound, so the section only appears when a play applies or the load failed.
export const PlaysRailSection = ({ ticket }: PlaysRailSectionProps) => {
  const canRun = useHasPermission("plays:run");
  const { data: plays = [], error } = useApplicableTicketPlays(ticket);
  const hasPlays = plays.length > 0;

  return (
    canRun &&
    (hasPlays || Boolean(error)) && (
      <section className="space-y-0.5">
        <h2 className={cn(microheaderClass, "px-2 pb-1")}>Plays</h2>
        {error && <ErrorDisplay error={error} title="Couldn't load plays." />}
        {hasPlays && <HarnessReadinessNote projectId={ticket.project_id} className="px-2 pb-1" />}
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
                className="w-full [&>button]:h-auto [&>button]:w-full [&>button]:justify-start [&>button]:gap-2 [&>button]:px-2! [&>button]:py-1.5 [&>button]:text-xs [&>button]:font-normal [&>span]:px-2"
              />
            ))}
          </div>
        )}
      </section>
    )
  );
};
