import { ServerIcon } from "lucide-react";

import { NoDataDisplay } from "@/components/NoDataDisplay";
import { RunnerRow } from "@/components/runner/RunnerRow";
import type { Runner } from "@/models/Runner";

interface RunnersSectionProps {
  runners: Runner[] | undefined;
}

export const RunnersSection = ({ runners }: RunnersSectionProps) => (
  <section className="space-y-3">
    <h2 className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-muted-foreground">
      <ServerIcon className="size-4" aria-hidden />
      Runners
    </h2>
    {runners && runners.length === 0 && <NoDataDisplay message="No runners connected yet." />}
    {runners && runners.length > 0 && (
      <ul className="divide-y divide-border rounded-lg border border-border bg-card">
        {runners.map((runner, index) => (
          <RunnerRow key={runner.id} runner={runner} index={index} />
        ))}
      </ul>
    )}
  </section>
);
