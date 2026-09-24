import { useMemo } from "react";

import { DocBodyView } from "@/components/doc/DocBodyView";
import type { Ticket } from "@/models/Ticket";
import { acceptanceCriteria } from "@/utils/AcceptanceCriteriaUtility";

interface AcceptanceCriteriaBlockProps {
  ticket: Ticket;
}

export const AcceptanceCriteriaBlock = ({ ticket }: AcceptanceCriteriaBlockProps) => {
  const criteria = useMemo(() => acceptanceCriteria(ticket.body), [ticket.body]);
  return (
    <div className="space-y-1">
      <h3 className="px-2 text-xs text-muted-foreground">What to check</h3>
      {criteria === "" && (
        <p role="status" className="px-2 text-sm text-muted-foreground">
          This ticket has no Acceptance criteria section. Check what its description asks for.
        </p>
      )}
      {criteria !== "" && (
        <div className="px-2">
          <DocBodyView key={criteria} body={criteria} />
        </div>
      )}
    </div>
  );
};
