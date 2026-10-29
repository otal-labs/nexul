import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { WaitingRow } from "@/components/runner/WaitingRow";
import type { QueuedJob, Runner } from "@/models/Runner";
import { queueReason } from "@/utils/RunnerFleetUtility";

interface WaitingSectionProps {
  queue: QueuedJob[];
  runners: Runner[];
}

const SHOWN = 5;

// Absent while nothing waits; each job says why it waits, read from the runners on the page.
export const WaitingSection = ({ queue, runners }: WaitingSectionProps) => {
  const [all, setAll] = useState(false);
  if (queue.length === 0) return null;
  const shown = all ? queue : queue.slice(0, SHOWN);
  return (
    <section className="space-y-2" aria-labelledby="runners-waiting">
      <Microheader id="runners-waiting">
        Waiting for a runner · <span className="tabular-nums">{queue.length}</span>
      </Microheader>
      <EnterList className="@container divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
        {shown.map((job, index) => (
          <WaitingRow key={job.id} job={job} position={index + 1} reason={queueReason(job, runners)} />
        ))}
      </EnterList>
      {queue.length > SHOWN && (
        <Button type="button" variant="link" size="sm" className="h-auto px-0 text-muted-foreground" onClick={() => setAll(!all)}>
          {all ? "Show fewer" : `Show ${queue.length - SHOWN} more`}
        </Button>
      )}
    </section>
  );
};
