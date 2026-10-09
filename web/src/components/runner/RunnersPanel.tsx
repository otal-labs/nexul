import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FleetStatRow } from "@/components/runner/FleetStatRow";
import { MachineGroup } from "@/components/runner/MachineGroup";
import { QueueSection } from "@/components/runner/QueueSection";
import { RunnersSection } from "@/components/runner/RunnersSection";
import { useFetchMachines, useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";
import type { Runner } from "@/models/Runner";

// Runners without a resolved machine (never connected, or connected before machines existed) fall back to a
// flat list — there's nothing to group them under yet.
const groupByMachine = (runners: Runner[]) => {
  const byMachine = new Map<string, Runner[]>();
  const unassigned: Runner[] = [];
  for (const runner of runners) {
    if (!runner.machine) {
      unassigned.push(runner);
      continue;
    }
    byMachine.set(runner.machine, [...(byMachine.get(runner.machine) ?? []), runner]);
  }
  return { byMachine, unassigned };
};

export const RunnersPanel = () => {
  const runners = useRunners();
  const machines = useFetchMachines();
  const queue = useRunnerQueue();
  const onlineCount = runners.data?.filter((runner) => runner.connected).length ?? 0;
  const { byMachine, unassigned } = groupByMachine(runners.data ?? []);

  return (
    <div className="space-y-6">
      {(runners.isPending || queue.isPending || machines.isPending) && <LoadingDisplay />}
      {(runners.isError || queue.isError || machines.isError) && (
        <ErrorDisplay error={runners.error ?? queue.error ?? machines.error} title="Couldn't load runners." />
      )}
      {runners.data && queue.data && (
        <FleetStatRow
          online={onlineCount}
          busy={runners.data.filter((runner) => runner.running_job).length}
          offline={runners.data.length - onlineCount}
          queued={queue.data.length}
        />
      )}
      <div className="grid gap-8 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div className="min-w-0 space-y-8">
          {machines.data && machines.data.length === 0 && <RunnersSection runners={runners.data} />}
          {machines.data && machines.data.length > 0 && (
            <div className="space-y-4">
              {machines.data.map((machine) => (
                <MachineGroup key={machine.id} machine={machine} runners={byMachine.get(machine.name) ?? []} />
              ))}
            </div>
          )}
          {machines.data && machines.data.length > 0 && unassigned.length > 0 && <RunnersSection runners={unassigned} />}
        </div>
        <QueueSection queue={queue.data} />
      </div>
    </div>
  );
};
