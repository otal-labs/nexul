import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlayButton } from "@/components/play/PlayButton";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { Ticket } from "@/models/Ticket";

interface PlaysRailSectionProps {
  ticket: Ticket;
}

const microheaderClass =
  "px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Plays are stage-bound, so a ticket in a stage none targets keeps the section with a sentence rather than losing it.
export const PlaysRailSection = ({ ticket }: PlaysRailSectionProps) => {
  const canRun = useHasPermission("plays:run");
  const { data: plays, error, isLoading } = useApplicableTicketPlays(ticket);

  return (
    canRun && (
      <section className="space-y-0.5">
        <h2 className={microheaderClass}>Plays</h2>
        {isLoading && <LoadingDisplay label="Loading plays…" />}
        {error && <ErrorDisplay error={error} title="Failed to load plays." />}
        {plays?.length === 0 && <p className="px-2 text-xs text-muted-foreground">No plays for this stage.</p>}
        {plays && plays.length > 0 && (
          <div className="flex flex-col gap-1 px-2">
            {plays.map((play) => (
              <PlayButton
                key={play.id}
                play={play}
                projectId={ticket.project_id}
                targetType="ticket"
                targetId={ticket.id}
                variant="ghost"
                className="w-full [&>button]:w-full [&>button]:justify-start"
              />
            ))}
          </div>
        )}
      </section>
    )
  );
};
