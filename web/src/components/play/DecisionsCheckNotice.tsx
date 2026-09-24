import { RotateCw, TriangleAlert } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useMissedDecisionsCheck, useRunDecisionsCheck } from "@/hooks/TrailHooks";

interface DecisionsCheckNoticeProps {
  ticketId: string;
}

// Shown only while the ticket's latest decisions check failed, so a check that could not start is never silent.
export const DecisionsCheckNotice = ({ ticketId }: DecisionsCheckNoticeProps) => {
  const missed = useMissedDecisionsCheck(ticketId);
  const run = useRunDecisionsCheck();
  if (!missed) return null;

  return (
    <section
      aria-label="Decisions check"
      className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-md border border-border px-3 py-2"
    >
      <TriangleAlert className="size-4 shrink-0 text-warning" aria-hidden />
      <div className="min-w-0 flex-1 basis-48">
        <p className="text-sm">Decisions check didn't run</p>
        {missed.last_error && <p className="text-xs break-words text-muted-foreground">{missed.last_error}</p>}
      </div>
      <Button variant="outline" size="sm" disabled={run.isPending} onClick={() => run.mutate(ticketId)}>
        <RotateCw aria-hidden />
        Run check
      </Button>
    </section>
  );
};
