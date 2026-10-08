import { WrenchIcon } from "lucide-react";

import { NoDataDisplay } from "@/components/NoDataDisplay";
import { QueueRow } from "@/components/runner/QueueRow";
import type { QueuedJob } from "@/models/Runner";

interface QueueSectionProps {
  queue: QueuedJob[] | undefined;
}

export const QueueSection = ({ queue }: QueueSectionProps) => (
  <section className="space-y-3">
    <h2 className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
      <WrenchIcon className="size-4" aria-hidden />
      Waiting for a runner
    </h2>
    {queue && queue.length === 0 && <NoDataDisplay message="Nothing queued." />}
    {queue && queue.length > 0 && (
      <ul className="divide-y divide-border rounded-lg border border-border bg-card">
        {queue.map((job, index) => (
          <QueueRow key={job.id} job={job} index={index} position={index + 1} />
        ))}
      </ul>
    )}
  </section>
);
