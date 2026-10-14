import { Check, X } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
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

// Pass and Fail take tickets:write, like any other change to the ticket.
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
            <ActionButton permission="tickets:write" projectId={ticket.project_id} icon={Check} size="sm" className="flex-1" loading={pass.isPending} onClick={() => pass.mutate(ticket.id)}>
              Pass
            </ActionButton>
            <ActionButton permission="tickets:write" projectId={ticket.project_id} icon={X} size="sm" variant="outline" className="flex-1" onClick={() => void openFail(ticket.id)}>
              Fail
            </ActionButton>
          </div>
        </div>
      )}
    </section>
  );
};
