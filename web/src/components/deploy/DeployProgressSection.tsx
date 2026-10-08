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
  const emptyMessage = terminal ? "No output recorded." : "Waiting for the runner to pick this up…";

  return (
    <section
      aria-label="Progress"
      // The steps run down beside the log from a 48rem page, held in view while the log scrolls.
      className="grid gap-6 pt-6 @3xl:grid-cols-[15rem_minmax(0,1fr)] @3xl:gap-10"
    >
      <div className="@3xl:sticky @3xl:top-6 @3xl:self-start @3xl:pt-12">
        <DeployStepList title={progress.title} steps={progress.steps} />
      </div>
      <div className="min-w-0 space-y-2">
        <DeployLogActions deployId={deploy.id} lines={lines ?? []} />
        {isPending && <LoadingDisplay label="Loading log" />}
        {error && <ErrorDisplay error={error} title="Couldn't load the log" />}
        {lines && <DeployLogPanel lines={lines} emptyMessage={emptyMessage} />}
      </div>
      {!terminal && (
        <div className="flex justify-end @3xl:col-start-2">
          <DeployCancelButton deployId={deploy.id} />
        </div>
      )}
    </section>
  );
};
