import { PlayButton } from "@/components/play/PlayButton";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import type { Ticket } from "@/models/Ticket";

interface PlaysRailSectionProps {
  ticket: Ticket;
}

const microheaderClass =
  "px-2 pb-1 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

// Renders nothing until a play applies to the ticket's current stage; a rail section with no buttons is noise.
export const PlaysRailSection = ({ ticket }: PlaysRailSectionProps) => {
  const { data: plays } = useApplicableTicketPlays(ticket);
  if (!plays || plays.length === 0) return null;

  return (
    <section className="space-y-0.5">
      <h2 className={microheaderClass}>Plays</h2>
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
    </section>
  );
};
