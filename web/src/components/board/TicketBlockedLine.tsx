import { Ban } from "lucide-react";

import { useBoardCardData } from "@/hooks/useBoardCardData";
import { linkedTicketKey } from "@/models/TicketLink";

interface TicketBlockedLineProps {
  ticketId: string;
}

// A signal, never a gate: the card still drags anywhere while this shows.
export const TicketBlockedLine = ({ ticketId }: TicketBlockedLineProps) => {
  const { blockers } = useBoardCardData();
  const waitingOn = blockers?.[ticketId] ?? [];
  if (waitingOn.length === 0) return null;

  const keys = waitingOn.map(linkedTicketKey).join(", ");

  // The blocking key, never truncated, on the card's meta line; the sentence is its tooltip and its accessible name.
  return (
    <span role="img" aria-label={`Blocked by ${keys}`} title={`Blocked by ${keys}`} className="flex shrink-0 items-baseline gap-1">
      <Ban className="size-3 shrink-0 self-center text-warning" aria-hidden />
      {keys}
    </span>
  );
};
