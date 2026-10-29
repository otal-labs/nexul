import { zodResolver } from "@hookform/resolvers/zod";
import { HammerIcon, RefreshCwIcon, RocketIcon } from "lucide-react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { FormInput } from "@/components/FormInput";
import { MonoRef } from "@/components/MonoRef";
import { ActionButton } from "@/components/action/ActionButton";
import { LockedReason } from "@/components/action/LockedReason";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { useDeployStack, useRollbackStack } from "@/hooks/StackHooks";
import { useStackActions } from "@/hooks/useStackActions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployRefFormSchema, deployPath, type Deploy, type DeployRefFormData, type Stack } from "@/models/Stack";
import { imageShortName } from "@/utils/ImageRefUtility";

interface StackDeployButtonProps {
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
        <SettingsHint>Pulls the image again and restarts the container.</SettingsHint>
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

const NO_WAY = "This stack has no repository to build from and nothing running to redeploy.";

// One way to get the next version per stack shape: build from a ref (repo attached), redeploy the running image
// (run stack, no repo), or nothing until a repository is attached (compose always deploys from its repo); the way
// back, Rollback, sits under it. The page's one primary action, so it has a fixed seat in the header.
export const StackDeployButton = ({ stack, lastHealthy, image }: StackDeployButtonProps) => {
  const deploy = useDeployStack();
  const rollback = useRollbackStack();
  const answer = useStackActions(stack);
  const canBuild = stack.build_source != null;
  const canRedeploy = !canBuild && stack.strategy === "run" && !!image;
  // A build leaves no image and the server refuses a rollback without one.
  const rollbackImage = lastHealthy?.image;
  const nothing = !canBuild && !canRedeploy && !rollbackImage;

  return (
    <>
      {nothing && (
        <LockedReason reason={NO_WAY}>
          <Button type="button" size="sm" aria-disabled onClick={(event) => event.preventDefault()} className="cursor-default bg-foreground/[0.06] text-muted-foreground hover:bg-foreground/[0.06]">
            <RocketIcon /> Deploy
          </Button>
        </LockedReason>
      )}
      {!nothing && (
        <Popover>
          <PopoverTrigger asChild>
            <ActionButton
              permission="deploys:write"
              projectId={stack.project_id || undefined}
              answer={answer("deploys:write")}
              verb="deploy"
              type="button"
              size="sm"
              icon={RocketIcon}
            >
              Deploy
            </ActionButton>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-96 space-y-3 p-4">
            {canBuild && <BuildRefForm stack={stack} deploy={deploy} />}
            {canRedeploy && image && <RedeployRow stack={stack} image={image} deploy={deploy} />}
            {!canBuild && !canRedeploy && <SettingsHint>{NO_WAY}</SettingsHint>}
            {rollbackImage && (
              <div className="flex items-center justify-between gap-3 border-t border-border pt-3">
                <p className="min-w-0 flex-1 font-mono text-xs text-muted-foreground" title={rollbackImage}>
                  <span className="block font-sans">Last healthy image</span>
                  <MonoRef value={imageShortName(rollbackImage)} kind="image" title={rollbackImage} />
                </p>
                <RollbackButton stack={stack} image={rollbackImage} deployPending={deploy.isPending} rollback={rollback} />
              </div>
            )}
          </PopoverContent>
        </Popover>
      )}
    </>
  );
};
