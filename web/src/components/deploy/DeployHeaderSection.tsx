import { GitBranchIcon, GitCommitHorizontalIcon } from "lucide-react";

import { PageHeader } from "@/components/PageHeader";
import type { Crumb } from "@/components/PageBreadcrumb";
import { DeployCancelButton } from "@/components/deploy/DeployCancelButton";
import { DeployRetryButton } from "@/components/deploy/DeployRetryButton";
import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useActorNamer } from "@/hooks/usePersonNamer";
import { useFetchStack } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { deployCommit, type Deploy } from "@/models/Stack";
import { deployTitle, isTerminal } from "@/utils/DeployLogUtility";
import { imageShortName } from "@/utils/ImageRefUtility";

interface DeployHeaderSectionProps {
  deploy: Deploy;
}

// The record is what was deployed (the image's tag, else the ref or commit); the outcome is the meta line's status.
const recordTitle = (deploy: Deploy, commit: string): string => {
  if (deploy.image) return imageShortName(deploy.image);
  if (deploy.ref && deploy.ref !== commit) return deploy.ref;
  if (commit) return commit.slice(0, 7);
  return deployTitle(deploy);
};

export const DeployHeaderSection = ({ deploy }: DeployHeaderSectionProps) => {
  const wsPath = useWorkspacePath();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(stack?.project_id || undefined);
  const canOpenStack = useAreaAccess(stack?.project_id || undefined)?.("stacks") ?? false;
  const actor = useActorNamer()(deploy.triggered_by);
  const commit = deployCommit(deploy);
  const title = recordTitle(deploy, commit);
  // A branch that is already the title is not repeated; a commit build names the branch it came from.
  const branch = deploy.ref ? (deploy.ref === commit || deploy.ref === title ? undefined : deploy.ref) : stack?.build_source?.branch;
  const stackPath = wsPath(`/stacks/${deploy.stack_id}`);
  const crumbs: Crumb[] = [
    workspaceCrumb,
    ...(projectCrumb ? [projectCrumb] : []),
    ...(stack && canOpenStack
      ? [
          { label: stack.name, to: stackPath },
          { label: "Deploys", to: `${stackPath}/deploys` },
        ]
      : []),
  ];

  return (
    <PageHeader
      crumbs={crumbs}
      title={title}
      actions={
        <>
          {!isTerminal(deploy) && <DeployCancelButton deploy={deploy} />}
          <DeployRetryButton deploy={deploy} />
        </>
      }
      meta={
        <>
          <DeployStatusBadge status={deploy.status} className="text-foreground" />
          <span className="font-mono text-xs wrap-anywhere">{deploy.image || "repo build"}</span>
          {!deploy.image && branch && (
            <span className="inline-flex items-center gap-1 font-mono text-xs wrap-anywhere">
              <GitBranchIcon className="size-3 shrink-0" aria-hidden />
              {branch}
            </span>
          )}
          {commit && title !== commit.slice(0, 7) && (
            <span className="inline-flex items-center gap-1 font-mono text-xs" title={commit}>
              <GitCommitHorizontalIcon className="size-3 shrink-0" aria-hidden />
              {commit.slice(0, 7)}
            </span>
          )}
          {actor && <span className="font-mono text-xs">{actor}</span>}
          <time dateTime={deploy.created_at} title={new Date(deploy.created_at).toLocaleString()} className="font-mono text-xs">
            {formatRelativeTime(deploy.created_at)}
          </time>
        </>
      }
    />
  );
};
