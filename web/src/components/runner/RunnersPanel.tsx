import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { FleetFeed } from "@/components/runner/FleetFeed";
import { WaitingSection } from "@/components/runner/WaitingSection";
import { useFetchMachines, useOpenEnrollments, useRunnerQueue, useRunners } from "@/hooks/RunnerHooks";
import { groupFleet } from "@/utils/RunnerFleetUtility";

export const RunnersPanel = () => {
  const runners = useRunners();
  const machines = useFetchMachines();
  const queue = useRunnerQueue();
  const { data: codes } = useOpenEnrollments(false);
  const pending = runners.isPending || queue.isPending || machines.isPending;
  const error = runners.error ?? queue.error ?? machines.error;
  const groups = runners.data && machines.data && groupFleet(machines.data, runners.data, codes ?? []);

  return (
    <div className="space-y-6">
      {pending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load runners." />}
      {queue.data && runners.data && <WaitingSection queue={queue.data} runners={runners.data} />}
      {groups && groups.length === 0 && <NoDataDisplay message="No runners on the machines you can see yet." />}
      {groups && groups.length > 0 && <FleetFeed groups={groups} />}
    </div>
  );
};
