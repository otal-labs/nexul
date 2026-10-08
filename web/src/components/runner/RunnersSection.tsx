import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { RunnerRow } from "@/components/runner/RunnerRow";
import type { Runner } from "@/models/Runner";

interface RunnersSectionProps {
  runners: Runner[] | undefined;
}

export const RunnersSection = ({ runners }: RunnersSectionProps) => (
  <section className="space-y-3">
    <Microheader>Runners</Microheader>
    {runners && runners.length === 0 && <NoDataDisplay message="No runners yet. Add one to start building and deploying." />}
    {runners && runners.length > 0 && (
      <EnterList className="divide-y divide-border rounded-lg border border-border bg-card">
        {runners.map((runner) => (
          <RunnerRow key={runner.id} runner={runner} />
        ))}
      </EnterList>
    )}
  </section>
);
