import { Ban } from "lucide-react";
import { Fragment } from "react";

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

  const keys = waitingOn.map(linkedTicketKey);

  // Every key shows whole, wrapping between keys when they run long: a cut key names no ticket.
  return (
    <span title={`Blocked by ${keys.join(", ")}`} className="flex min-w-0 items-start gap-1.5 text-xs text-muted-foreground">
      <Ban className="mt-[1.5px] size-3 shrink-0 text-warning" aria-hidden />
      <span className="min-w-0">
        Blocked by{" "}
        {keys.map((key, index) => (
          <Fragment key={key}>
            {index > 0 && ", "}
            <span className="whitespace-nowrap font-mono">{key}</span>
          </Fragment>
        ))}
      </span>
    </span>
  );
};
