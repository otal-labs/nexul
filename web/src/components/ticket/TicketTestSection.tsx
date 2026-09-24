import { Check, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { AcceptanceCriteriaBlock } from "@/components/ticket/AcceptanceCriteriaBlock";
import { TestTargetRow } from "@/components/ticket/TestTargetRow";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useTestPass } from "@/hooks/TicketTestHooks";
import { useTestFailDialog } from "@/hooks/useTestFailDialog";
import { StatusKind } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

const microheaderClass =
  "px-2 font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase";

interface TicketTestSectionProps {
  ticket: Ticket;
}

// Sits in the main column so testers on a phone see it; anyone who can see the ticket may pass or fail it.
export const TicketTestSection = ({ ticket }: TicketTestSectionProps) => {
  const { data: statuses } = useFetchProjectStatuses(ticket.project_id);
  const pass = useTestPass();
  const openFail = useTestFailDialog();
  const testing = statuses?.some((s) => s.id === ticket.status && s.kind === StatusKind.Testing) ?? false;
  if (!testing) return null;

  return (
    <section aria-labelledby="test-this" className="space-y-4 border-t border-border pt-6">
      <h2 id="test-this" className={microheaderClass}>
        Test this
      </h2>
      <TestTargetRow ticket={ticket} />
      <AcceptanceCriteriaBlock ticket={ticket} />
      <div className="flex gap-2 px-2">
        <Button className="flex-1 sm:flex-none" disabled={pass.isPending} onClick={() => pass.mutate(ticket.id)}>
          <Check className="size-4" aria-hidden />
          Pass
        </Button>
        <Button variant="outline" className="flex-1 sm:flex-none" onClick={() => void openFail(ticket.id)}>
          <X className="size-4" aria-hidden />
          Fail
        </Button>
      </div>
    </section>
  );
};
