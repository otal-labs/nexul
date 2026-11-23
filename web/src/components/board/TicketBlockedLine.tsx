import { Ban } from "lucide-react";

import { OverflowCount } from "@/components/OverflowCount";
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

  const [first, ...rest] = waitingOn.map(linkedTicketKey);

  // The first key and the count stay whole and may wrap under "Blocked by", so the line is two rows at most; the count lists the rest.
  return (
    <span className="flex min-w-0 items-start gap-1.5 text-xs text-muted-foreground">
      <Ban className="mt-[1.5px] size-3 shrink-0 text-warning" aria-hidden />
      <span className="min-w-0">
        Blocked by{" "}
        <span className="whitespace-nowrap font-mono">
          {first}
          {rest.length > 0 && " "}
          {rest.length > 0 && <OverflowCount label={`${rest.length} more ${rest.length === 1 ? "blocker" : "blockers"}`} items={rest} />}
        </span>
      </span>
    </span>
  );
};
