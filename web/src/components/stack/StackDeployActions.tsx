import { zodResolver } from "@hookform/resolvers/zod";
import { HammerIcon, RefreshCwIcon, RocketIcon } from "lucide-react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";

import { FormInput } from "@/components/FormInput";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useDeployStack, useRollbackStack } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployRefFormSchema, deployPath, type Deploy, type DeployRefFormData, type Stack } from "@/models/Stack";

interface StackDeployActionsProps {
  stack: Stack;
  lastHealthy: Deploy | undefined;
  canRollback: boolean;
  // The image this stack currently runs: the latest deploy's, else what the runner observed.
  image: string | undefined;
}

type DeployMutation = ReturnType<typeof useDeployStack>;

// A compose deploy records no single image, so it is named by when it ran instead.
const rollbackTarget = (d: Deploy): string => d.image || `the deploy from ${formatRelativeTime(d.created_at)}`;
type RollbackMutation = ReturnType<typeof useRollbackStack>;

interface BuildRefFormProps {
  stack: Stack;
  deploy: DeployMutation;
}

// Build triggers are UI-only; MCP never triggers builds.
const BuildRefForm = ({ stack, deploy }: BuildRefFormProps) => {
  const form = useForm<DeployRefFormData>({
    defaultValues: { ref: stack.build_source?.branch ?? "" },
    resolver: zodResolver(DeployRefFormSchema),
  });
  const isDeploying = deploy.isPending && !!deploy.variables?.ref;
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();

  const onSubmit = async ({ ref }: DeployRefFormData) => {
    try {
      const created = await deploy.mutateAsync({ stackId: stack.id, ref });
      await navigate(wsPath(deployPath(created)));
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <form className="flex flex-wrap items-end gap-2" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="min-w-0 flex-1 basis-64">
        <FormInput control={form.control} name="ref" label="Build & deploy ref" placeholder={stack.build_source?.branch ?? "main"} />
      </div>
      <Button type="submit" loading={isDeploying} disabled={deploy.isPending}>
        <HammerIcon className="size-4" aria-hidden />
        {isDeploying ? "Building & deploying…" : "Build & deploy"}
      </Button>
    </form>
  );
};

interface RedeployRowProps {
  stack: Stack;
  image: string;
  deploy: DeployMutation;
}

// The runner's pre-built-image path only knows the run strategy: it removes the container and runs the image again.
const RedeployRow = ({ stack, image, deploy }: RedeployRowProps) => {
  const isDeploying = deploy.isPending && !!deploy.variables?.image;
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0 space-y-1">
        <p className="font-mono text-sm wrap-anywhere">{image}</p>
        <p className="text-xs text-muted-foreground">Pulls the image again and restarts the container.</p>
      </div>
      <Button
        type="button"
        loading={isDeploying}
        disabled={deploy.isPending}
        onClick={() => deploy.mutate({ stackId: stack.id, image }, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
      >
        <RocketIcon className="size-4" aria-hidden />
        {isDeploying ? "Redeploying…" : "Redeploy"}
      </Button>
    </div>
  );
};

interface RollbackButtonProps {
  stack: Stack;
  lastHealthy: Deploy | undefined;
  canRollback: boolean;
  deployPending: boolean;
  rollback: RollbackMutation;
}

const RollbackButton = ({ stack, lastHealthy, canRollback, deployPending, rollback }: RollbackButtonProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={() => rollback.mutate(stack.id, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
      loading={rollback.isPending}
      disabled={!canRollback || deployPending}
      title={lastHealthy ? `Roll back to ${rollbackTarget(lastHealthy)}` : "No healthy deploy to roll back to"}
    >
      <RefreshCwIcon className="size-4" aria-hidden />
      {rollback.isPending ? "Rolling back…" : "Rollback"}
    </Button>
  );
};

// One way to get the next version per stack shape: build from a ref (repo attached), redeploy the running image
// (run stack, no repo), or nothing until a repository is attached (compose always deploys from its repo).
export const StackDeployActions = ({ stack, lastHealthy, canRollback, image }: StackDeployActionsProps) => {
  const deploy = useDeployStack();
  const rollback = useRollbackStack();
  const canBuild = stack.build_source != null;
  const canRedeploy = !canBuild && stack.strategy === "run" && !!image;

  return (
    <SettingsCard
      id="deploy"
      title="Deploy"
      description="How this stack gets its next version."
      footer={
        <>
          <p className="min-w-0 flex-1 text-xs break-words text-muted-foreground">
            {lastHealthy?.image && `Rollback re-deploys ${lastHealthy.image}, the last healthy image.`}
            {lastHealthy && !lastHealthy.image && `Rollback re-deploys ${rollbackTarget(lastHealthy)}, the last healthy one.`}
            {!canRollback && "Rollback needs a healthy deploy in this stack's history."}
          </p>
          <RollbackButton
            stack={stack}
            lastHealthy={lastHealthy}
            canRollback={canRollback}
            deployPending={deploy.isPending}
            rollback={rollback}
          />
        </>
      }
    >
      {canBuild && <BuildRefForm stack={stack} deploy={deploy} />}
      {canRedeploy && image && <RedeployRow stack={stack} image={image} deploy={deploy} />}
      {!canBuild && !canRedeploy && (
        <p className="text-sm text-muted-foreground">
          Attach a repository to build and deploy this stack from here.
        </p>
      )}
    </SettingsCard>
  );
};
