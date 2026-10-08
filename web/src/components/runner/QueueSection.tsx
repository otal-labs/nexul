import { EmptyRow } from "@/components/EmptyRow";
import { Microheader } from "@/components/Microheader";
import { QueueRow } from "@/components/runner/QueueRow";
import type { QueuedJob } from "@/models/Runner";

interface QueueSectionProps {
  queue: QueuedJob[] | undefined;
}

export const QueueSection = ({ queue }: QueueSectionProps) => (
  <section className="space-y-3">
    <Microheader>Waiting for a runner</Microheader>
    {queue && queue.length === 0 && <EmptyRow flush>Nothing queued.</EmptyRow>}
    {queue && queue.length > 0 && (
      <ul className="divide-y divide-border rounded-lg border border-border bg-card">
        {queue.map((job, index) => (
          <QueueRow key={job.id} job={job} index={index} position={index + 1} />
        ))}
      </ul>
    )}
  </section>
);
