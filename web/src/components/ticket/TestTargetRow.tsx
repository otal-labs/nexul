import { ExternalLink } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoTestTargetRow } from "@/components/ticket/NoTestTargetRow";
import { useFetchTestTarget } from "@/hooks/TicketTestHooks";
import type { Ticket } from "@/models/Ticket";
import { TestTargetKind } from "@/models/TicketTest";

interface TestTargetRowProps {
  ticket: Ticket;
}

export const TestTargetRow = ({ ticket }: TestTargetRowProps) => {
  const { data: target, error, isPending } = useFetchTestTarget(ticket.id);
  return (
    <div className="space-y-1">
      <h3 className="px-2 text-xs text-muted-foreground">Where to test</h3>
      {isPending && <LoadingDisplay label="Finding a test environment…" className="p-3" />}
      {error && <ErrorDisplay error={error} title="Failed to find a test environment." />}
      {target && target.url === "" && <NoTestTargetRow projectId={ticket.project_id} />}
      {target && target.url !== "" && (
        <div className="px-2">
          <a
            href={target.url}
            target="_blank"
            rel="noreferrer"
            className="inline-flex max-w-full items-center gap-2 py-1 font-mono text-sm break-all underline-offset-4 hover:underline"
          >
            <ExternalLink className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
            {target.url}
          </a>
          {target.kind === TestTargetKind.Shared && (
            <p className="text-xs text-muted-foreground">Shared, may include other changes.</p>
          )}
          {target.kind === TestTargetKind.Preview && target.branch && (
            <p className="text-xs text-muted-foreground">
              Preview of <span className="font-mono">{target.branch}</span>
            </p>
          )}
        </div>
      )}
    </div>
  );
};
