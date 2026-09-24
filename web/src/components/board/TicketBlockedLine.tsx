import { Ban } from "lucide-react";

import { useFetchBlockers } from "@/hooks/TicketLinkHooks";
import { linkedTicketKey } from "@/models/TicketLink";

interface TicketBlockedLineProps {
  ticketId: string;
}

// A signal, never a gate: the card still drags anywhere while this shows.
export const TicketBlockedLine = ({ ticketId }: TicketBlockedLineProps) => {
  const { data: blockers } = useFetchBlockers();
  const waitingOn = blockers?.[ticketId] ?? [];
  if (waitingOn.length === 0) return null;

  return (
    <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
      <Ban className="size-3 shrink-0 text-warning" aria-hidden />
      <span className="truncate">
        Blocked by <span className="font-mono">{waitingOn.map(linkedTicketKey).join(", ")}</span>
      </span>
    </span>
  );
};
