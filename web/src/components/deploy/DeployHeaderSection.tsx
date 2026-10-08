import { GitBranchIcon } from "lucide-react";

import { PageHeader } from "@/components/PageHeader";
import type { Crumb } from "@/components/PageBreadcrumb";
import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useFetchStack } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Deploy } from "@/models/Stack";
import { deployTitle } from "@/utils/DeployLogUtility";

interface DeployHeaderSectionProps {
  deploy: Deploy;
}

export const DeployHeaderSection = ({ deploy }: DeployHeaderSectionProps) => {
  const wsPath = useWorkspacePath();
  const { data: stack } = useFetchStack(deploy.stack_id);
  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(stack?.project_id || undefined);
  const canOpenStack = useAreaAccess(stack?.project_id || undefined)?.("stacks") ?? false;
  const branch = stack?.build_source?.branch;
  const stackPath = wsPath(`/stacks/${deploy.stack_id}`);
  const crumbs: Crumb[] = [
    workspaceCrumb,
    ...(projectCrumb ? [projectCrumb] : []),
    ...(stack && canOpenStack
      ? [
          { label: stack.name, to: stackPath },
          { label: "Deploys", to: `${stackPath}/history` },
        ]
      : []),
  ];

  return (
    <PageHeader
      crumbs={crumbs}
      title={deployTitle(deploy)}
      meta={
        <>
          <DeployStatusBadge status={deploy.status} />
          <span className="font-mono text-xs wrap-anywhere">{deploy.id}</span>
          <span className="font-mono text-xs wrap-anywhere">{deploy.image || "repo build"}</span>
          {!deploy.image && branch && (
            <span className="inline-flex items-center gap-1 font-mono text-xs wrap-anywhere">
              <GitBranchIcon className="size-3 shrink-0" aria-hidden />
              {branch}
            </span>
          )}
          <time dateTime={deploy.created_at}>{formatRelativeTime(deploy.created_at)}</time>
        </>
      }
    />
  );
};
