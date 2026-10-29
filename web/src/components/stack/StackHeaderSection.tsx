import { GitBranchIcon } from "lucide-react";

import { PageHeader } from "@/components/PageHeader";
import type { Crumb } from "@/components/PageBreadcrumb";
import { RowActions } from "@/components/listpane/RowActions";
import { StackDeployButton } from "@/components/stack/StackDeployButton";
import { StackLiveHero } from "@/components/stack/StackLiveHero";
import { StackMetaLine } from "@/components/stack/StackMetaLine";
import { StackWatchButton } from "@/components/stack/StackWatchButton";
import { ActionButton } from "@/components/action/ActionButton";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useStackActions } from "@/hooks/useStackActions";
import { useStackPermissions } from "@/hooks/useStackPermissions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Deploy, Stack, StackWithBranches } from "@/models/Stack";

interface StackHeaderSectionProps {
  stack: StackWithBranches;
  latest: Deploy | undefined;
  lastHealthy: Deploy | undefined;
  image: string | undefined;
  hostnames: string[];
}

// An instance stack (a gateway) has no project; Topology is where it lives.
const useStackCrumbs = (stack: Stack): Crumb[] => {
  const workspaceCrumb = useWorkspaceCrumb();
  const projectCrumb = useProjectCrumb(stack.project_id || undefined);
  const canOpenTopology = useAreaAccess()?.("topology") ?? false;
  const wsPath = useWorkspacePath();
  if (stack.project_id) return projectCrumb ? [workspaceCrumb, projectCrumb] : [workspaceCrumb];
  if (canOpenTopology) return [workspaceCrumb, { label: "Topology", to: wsPath("/topology") }];
  return [workspaceCrumb];
};

export const StackHeaderSection = ({ stack, latest, lastHealthy, image, hostnames }: StackHeaderSectionProps) => {
  const wsPath = useWorkspacePath();
  const answer = useStackActions(stack);
  const crumbs = useStackCrumbs(stack);
  const { menu, manages } = useStackPermissions(stack);

  return (
    <div className="space-y-6">
      <PageHeader
        className="border-b-0 pb-0"
        crumbs={crumbs}
        title={stack.name}
        meta={<StackMetaLine stack={stack} image={image} />}
        actions={
          <div className="flex items-center gap-2">
            {!stack.managed && !!stack.project_id && (
              <ActionButton
                permission={["stacks:write", "projects:write"]}
                projectId={stack.project_id}
                answer={answer("stacks:write")}
                verb="attach repositories"
                variant="outline"
                size="sm"
                icon={GitBranchIcon}
                to={wsPath(`/wizard/project/repository?stack=${stack.id}`)}
              >
                Attach repository
              </ActionButton>
            )}
            <StackDeployButton stack={stack} lastHealthy={lastHealthy} image={image} />
            <StackWatchButton stackId={stack.id} />
            {manages && <RowActions itemLabel={stack.name} {...menu} />}
          </div>
        }
      />

      <StackLiveHero stack={stack} latest={latest} lastHealthy={lastHealthy} hostnames={hostnames} />
    </div>
  );
};
