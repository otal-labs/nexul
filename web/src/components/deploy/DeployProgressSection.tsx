import { useMemo } from "react";

import { DeployCancelButton } from "@/components/deploy/DeployCancelButton";
import { DeployLogActions } from "@/components/deploy/DeployLogActions";
import { DeployLogPanel } from "@/components/deploy/DeployLogPanel";
import { DeployStepList } from "@/components/deploy/DeployStepList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDeployLog } from "@/hooks/DeployHooks";
import { useNow } from "@/hooks/useNow";
import type { Deploy } from "@/models/Stack";
import { deriveDeployProgress, isTerminal } from "@/utils/DeployLogUtility";

interface DeployProgressSectionProps {
  deploy: Deploy;
}

export const DeployProgressSection = ({ deploy }: DeployProgressSectionProps) => {
  const { data: lines, error, isPending } = useFetchDeployLog(deploy.id);
  const terminal = isTerminal(deploy);
  const now = useNow(!terminal);
  const progress = useMemo(() => deriveDeployProgress(deploy, lines ?? [], now), [deploy, lines, now]);
  const emptyMessage = terminal ? "No output was recorded." : "Waiting for the runner to pick this up…";

  return (
    <section className="space-y-6 pt-6">
      <h2 className="text-center text-xl font-semibold tracking-tight">{progress.title}</h2>
      <DeployStepList title={progress.title} steps={progress.steps} />
      <div className="space-y-2">
        <DeployLogActions deployId={deploy.id} lines={lines ?? []} />
        {isPending && <LoadingDisplay label="Loading log" />}
        {error && <ErrorDisplay error={error} title="Could not load the log" />}
        {lines && <DeployLogPanel lines={lines} emptyMessage={emptyMessage} />}
      </div>
      {!terminal && (
        <div className="flex justify-end">
          <DeployCancelButton deployId={deploy.id} />
        </div>
      )}
    </section>
  );
};
