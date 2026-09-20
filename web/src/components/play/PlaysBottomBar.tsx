import { PlayButton } from "@/components/play/PlayButton";
import { useApplicableTicketPlays } from "@/hooks/PlayHooks";
import type { Ticket } from "@/models/Ticket";

interface PlaysBottomBarProps {
  ticket: Ticket;
}

// The rail's plays when the rail itself is collapsed, so the page stays usable below the lg breakpoint.
export const PlaysBottomBar = ({ ticket }: PlaysBottomBarProps) => {
  const { data: plays } = useApplicableTicketPlays(ticket);
  if (!plays || plays.length === 0) return null;

  return (
    <div className="fixed inset-x-0 bottom-0 z-10 flex gap-2 overflow-x-auto border-t border-border bg-background/95 p-3 backdrop-blur lg:hidden">
      {plays.map((play) => (
        <PlayButton key={play.id} play={play} projectId={ticket.project_id} targetType="ticket" targetId={ticket.id} variant="default" />
      ))}
    </div>
  );
};
