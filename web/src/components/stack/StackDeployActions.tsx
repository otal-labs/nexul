import { zodResolver } from "@hookform/resolvers/zod";
import { HammerIcon, RefreshCwIcon, RocketIcon } from "lucide-react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";

import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ActionButton } from "@/components/action/ActionButton";
import { useDeployStack, useRollbackStack } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployRefFormSchema, deployPath, type Deploy, type DeployRefFormData, type Stack } from "@/models/Stack";

interface StackDeployActionsProps {
  stack: Stack;
  lastHealthy: Deploy | undefined;
  // The image this stack currently runs: the latest deploy's, else what the runner observed.
  image: string | undefined;
}

type DeployMutation = ReturnType<typeof useDeployStack>;

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
  const answer = useStackActions(stack);
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
      <ActionButton
        permission="deploys:write"
        projectId={stack.project_id || undefined}
        answer={answer("deploys:write")}
        verb="deploy"
        type="submit"
        icon={HammerIcon}
        loading={isDeploying}
        disabled={deploy.isPending}
      >
        {isDeploying ? "Building & deploying…" : "Build & deploy"}
      </ActionButton>
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
  const answer = useStackActions(stack);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0 space-y-1">
        <p className="font-mono text-sm wrap-anywhere">{image}</p>
        <p className="text-xs text-muted-foreground">Pulls the image again and restarts the container.</p>
      </div>
      <ActionButton
        permission="deploys:write"
        projectId={stack.project_id || undefined}
        answer={answer("deploys:write")}
        verb="deploy"
        type="button"
        icon={RocketIcon}
        loading={isDeploying}
        disabled={deploy.isPending}
        onClick={() => deploy.mutate({ stackId: stack.id, image }, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
      >
        {isDeploying ? "Redeploying…" : "Redeploy"}
      </ActionButton>
    </div>
  );
};

interface RollbackButtonProps {
  stack: Stack;
  image: string;
  deployPending: boolean;
  rollback: RollbackMutation;
}

const RollbackButton = ({ stack, image, deployPending, rollback }: RollbackButtonProps) => {
  const answer = useStackActions(stack);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  return (
    <ActionButton
      permission="deploys:write"
      projectId={stack.project_id || undefined}
      answer={answer("deploys:write")}
      verb="deploy"
      type="button"
      variant="outline"
      size="sm"
      icon={RefreshCwIcon}
      onClick={() => rollback.mutate(stack.id, { onSuccess: (created) => void navigate(wsPath(deployPath(created))) })}
      loading={rollback.isPending}
      disabled={deployPending}
      title={`Roll back to ${image}`}
    >
      {rollback.isPending ? "Rolling back…" : "Rollback"}
    </ActionButton>
  );
};

// One way to get the next version per stack shape: build from a ref (repo attached), redeploy the running image
// (run stack, no repo), or nothing until a repository is attached (compose always deploys from its repo).
export const StackDeployActions = ({ stack, lastHealthy, image }: StackDeployActionsProps) => {
  const deploy = useDeployStack();
  const rollback = useRollbackStack();
  const canBuild = stack.build_source != null;
  const canRedeploy = !canBuild && stack.strategy === "run" && !!image;
  // A build leaves no image and the server refuses a rollback without one.
  const rollbackImage = lastHealthy?.image;

  return (
    <SettingsCard
      id="deploy"
      title="Deploy"
      description="How this stack gets its next version."
      footer={
        rollbackImage && (
          <>
            <p className="min-w-0 flex-1 text-xs break-words text-muted-foreground">
              Rollback re-deploys {rollbackImage}, the last healthy image.
            </p>
            <RollbackButton stack={stack} image={rollbackImage} deployPending={deploy.isPending} rollback={rollback} />
          </>
        )
      }
    >
      {canBuild && <BuildRefForm stack={stack} deploy={deploy} />}
      {canRedeploy && image && <RedeployRow stack={stack} image={image} deploy={deploy} />}
      {!canBuild && !canRedeploy && (
        <p className="text-sm text-muted-foreground">
          This stack has no repository to build from and nothing running to redeploy, so it can't be deployed from here.
        </p>
      )}
    </SettingsCard>
  );
};
