import type { ReactNode } from "react";
import { GitBranchIcon } from "lucide-react";
import { Link } from "react-router";

import { Fact } from "@/components/Fact";
import { PageHeader } from "@/components/PageHeader";
import type { Crumb } from "@/components/PageBreadcrumb";
import { StackLiveHero } from "@/components/stack/StackLiveHero";
import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectCrumb, useWorkspaceCrumb } from "@/hooks/useCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Deploy, Stack } from "@/models/Stack";

interface StackHeaderSectionProps {
  stack: Stack;
  latest: Deploy | undefined;
  image: string | undefined;
  hostnames: string[];
}

// Image refs, paths and names here are identifiers someone copies, so they wrap instead of truncating.
const Mono = ({ children }: { children: ReactNode }) => (
  <span className="min-w-0 font-mono text-xs wrap-anywhere">{children}</span>
);

const repoLabel = (stack: Stack): string | undefined => {
  const src = stack.build_source;
  if (!src?.repo_owner || !src.repo_name) return undefined;
  return `${src.repo_owner}/${src.repo_name}`;
};

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

export const StackHeaderSection = ({ stack, latest, image, hostnames }: StackHeaderSectionProps) => {
  const can = useAreaAccess(stack.project_id || undefined);
  const wsPath = useWorkspacePath();
  const crumbs = useStackCrumbs(stack);
  const repo = repoLabel(stack);

  return (
    <div className="border-b border-border pb-6">
      <PageHeader
        className="border-b-0 pb-0"
        crumbs={crumbs}
        title={stack.name}
        meta={
          <>
            <span className="font-mono text-xs wrap-anywhere">{stack.slug}</span>
            {!stack.managed && <NoFillBadge color="bg-muted-foreground">unmanaged</NoFillBadge>}
          </>
        }
        actions={
          !stack.managed &&
          !!stack.project_id &&
          can?.("newProject") && (
            <Button variant="outline" size="sm" asChild>
              <Link to={wsPath(`/wizard/project/repository?stack=${stack.id}`)}>
                <GitBranchIcon className="size-4" /> Attach repository
              </Link>
            </Button>
          )
        }
      />

      <div className="@container mt-6">
        <StackLiveHero stackId={stack.id} latest={latest} hostnames={hostnames} />
        <dl className="mt-6 grid grid-cols-2 gap-x-8 gap-y-5 @xl:grid-cols-3">
          {stack.strategy !== "compose" && (
            <Fact label="Image">
              {image && <Mono>{image}</Mono>}
              {!image && <span className="text-muted-foreground">—</span>}
            </Fact>
          )}
          <Fact label="Machine">
            <Mono>{stack.machine}</Mono>
          </Fact>
          <Fact label="Strategy">
            <Mono>{stack.strategy}</Mono>
            {stack.compose_path && <Mono>{stack.compose_path}</Mono>}
          </Fact>
          {repo && (
            <Fact label="Repository">
              <Mono>{repo}</Mono>
              {stack.build_source?.branch && (
                <span className="inline-flex min-w-0 items-center gap-1 font-mono text-xs wrap-anywhere text-muted-foreground">
                  <GitBranchIcon className="size-3 shrink-0" aria-hidden />
                  {stack.build_source.branch}
                </span>
              )}
            </Fact>
          )}
          {!repo && (
            <Fact label="Network">
              {stack.docker_network && <Mono>{stack.docker_network}</Mono>}
              {!stack.docker_network && <span className="text-muted-foreground">—</span>}
            </Fact>
          )}
        </dl>
      </div>
    </div>
  );
};
