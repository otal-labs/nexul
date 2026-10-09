import { Check, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { microheaderClass } from "@/components/Microheader";
import { AcceptanceCriteriaBlock } from "@/components/ticket/AcceptanceCriteriaBlock";
import { TestTargetRow } from "@/components/ticket/TestTargetRow";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useTestPass } from "@/hooks/TicketTestHooks";
import { useTestFailDialog } from "@/hooks/useTestFailDialog";
import { StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";
import { cn } from "@/lib/utils";

interface TicketTestSectionProps {
  ticket: Ticket;
}

// Anyone who can see the ticket may pass or fail it.
export const TicketTestSection = ({ ticket }: TicketTestSectionProps) => {
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const pass = useTestPass();
  const openFail = useTestFailDialog();
  const testing = statuses?.some((s) => s.id === ticket.status && s.kind === StatusKind.Testing) ?? false;
  if (!statuses) return null;

  return (
    <section aria-labelledby="ticket-testing" className="space-y-2">
      <h2 id="ticket-testing" className={cn(microheaderClass, "px-2")}>
        Testing
      </h2>
      {!testing && <p className="px-2 text-xs text-muted-foreground">Pass and Fail show in a testing column.</p>}
      {testing && (
        <div className="space-y-3">
          <TestTargetRow ticket={ticket} />
          <AcceptanceCriteriaBlock ticket={ticket} />
          <div className="flex gap-2 px-2">
            <Button size="sm" className="flex-1" loading={pass.isPending} onClick={() => pass.mutate(ticket.id)}>
              <Check className="size-4" aria-hidden />
              Pass
            </Button>
            <Button size="sm" variant="outline" className="flex-1" onClick={() => void openFail(ticket.id)}>
              <X className="size-4" aria-hidden />
              Fail
            </Button>
          </div>
        </div>
      )}
    </section>
  );
};
