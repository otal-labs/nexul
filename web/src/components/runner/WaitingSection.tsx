import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { WaitingRow } from "@/components/runner/WaitingRow";
import type { QueuedJob, Runner } from "@/models/Runner";
import { cn } from "@/lib/utils";
import { queueReason } from "@/utils/RunnerFleetUtility";

interface WaitingSectionProps {
  queue: QueuedJob[];
  runners: Runner[];
}

const NARROW = 3;
const WIDE = 5;

// Absent while nothing waits; three rows below 44rem of list and five from there, then Show N more.
export const WaitingSection = ({ queue, runners }: WaitingSectionProps) => {
  const [all, setAll] = useState(false);
  if (queue.length === 0) return null;
  const shown = all ? queue : queue.slice(0, WIDE);
  return (
    <section className="@container space-y-2" aria-labelledby="runners-waiting">
      <Microheader id="runners-waiting">
        Waiting for a runner · <span className="tabular-nums">{queue.length}</span>
      </Microheader>
      <EnterList className="divide-y divide-border overflow-hidden rounded-lg border border-border bg-card">
        {shown.map((job, index) => (
          <WaitingRow key={job.id} job={job} position={index + 1} reason={queueReason(job, runners)} className={cn(!all && index >= NARROW && "@max-[44rem]:hidden")} />
        ))}
      </EnterList>
      {queue.length > NARROW && (
        <Button type="button" variant="link" size="sm" className={cn("h-auto px-0 text-muted-foreground", queue.length <= WIDE && !all && "@[44rem]:hidden")} onClick={() => setAll(!all)}>
          {all && "Show fewer"}
          {!all && <span className="@[44rem]:hidden">Show {queue.length - NARROW} more</span>}
          {!all && queue.length > WIDE && <span className="hidden @[44rem]:inline">Show {queue.length - WIDE} more</span>}
        </Button>
      )}
    </section>
  );
};
