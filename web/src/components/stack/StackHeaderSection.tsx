import type { ReactNode } from "react";
import { GitBranchIcon } from "lucide-react";
import { Link } from "react-router";

import { Fact } from "@/components/Fact";
import { DeployStatusBadge } from "@/components/service/DeployStatusBadge";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { ServicesFact } from "@/components/stack/ServicesFact";
import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import type { Deploy, Stack } from "@/models/Stack";

interface StackHeaderSectionProps {
  stack: Stack;
  projectPath: string;
  latest: Deploy | undefined;
  image: string | undefined;
  hostnames: string[];
}

const Mono = ({ children, title }: { children: ReactNode; title?: string }) => (
  <span className="min-w-0 truncate font-mono text-xs" title={title}>
    {children}
  </span>
);

const repoLabel = (stack: Stack): string | undefined => {
  const src = stack.build_source;
  if (!src?.repo_owner || !src.repo_name) return undefined;
  return `${src.repo_owner}/${src.repo_name}`;
};

// Detail-page header (back link → mono slug → title) plus the locked facts grid: mono microheader over each value.
export const StackHeaderSection = ({ stack, projectPath, latest, image, hostnames }: StackHeaderSectionProps) => {
  const can = useAreaAccess();
  const repo = repoLabel(stack);
  const [firstHostname, ...moreHostnames] = hostnames;

  return (
    <header className="space-y-5 border-b border-border pb-6">
      <div className="space-y-2">
        {can?.(stack.project_id ? "projects" : "topology") && (
          <Link
            to={projectPath}
            className="inline-block font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
          >
            {stack.project_id ? "← Back to project" : "← Back to topology"}
          </Link>
        )}
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-xs text-muted-foreground">{stack.slug}</span>
              {!stack.managed && <NoFillBadge color="bg-muted-foreground">unmanaged</NoFillBadge>}
            </div>
            <h1 className="text-3xl font-semibold tracking-tight">{stack.name}</h1>
          </div>
          {!stack.managed && !!stack.project_id && can?.("newProject") && (
            <Button variant="outline" size="sm" asChild>
              <Link to={`/wizard/project/repository?stack=${stack.id}`}>
                <GitBranchIcon className="size-4" /> Attach repository
              </Link>
            </Button>
          )}
        </div>
      </div>

      <dl className="grid grid-cols-2 gap-x-8 gap-y-5 sm:grid-cols-3">
        <Fact label="Status">
          {latest && <DeployStatusBadge status={latest.status} />}
          {latest && (
            <span className="text-xs text-muted-foreground">last deploy {formatRelativeTime(latest.created_at)}</span>
          )}
          {!latest && <span className="text-xs text-muted-foreground">No deploys yet</span>}
        </Fact>
        {stack.strategy === "compose" && (
          <Fact label="Services">
            <ServicesFact stackId={stack.id} />
          </Fact>
        )}
        {stack.strategy !== "compose" && (
          <Fact label="Image">
            {image && <Mono title={image}>{image}</Mono>}
            {!image && <span className="text-muted-foreground">—</span>}
          </Fact>
        )}
        <Fact label="Runner">
          <Mono>{stack.machine}</Mono>
        </Fact>
        <Fact label="Strategy">
          <Mono>{stack.strategy}</Mono>
          {stack.compose_path && <Mono title={stack.compose_path}>{stack.compose_path}</Mono>}
        </Fact>
        <Fact label="Hostnames">
          {firstHostname && (
            <a
              href={`https://${firstHostname}`}
              target="_blank"
              rel="noreferrer"
              className="min-w-0 truncate font-mono text-xs underline underline-offset-2"
            >
              {firstHostname}
            </a>
          )}
          {moreHostnames.length > 0 && (
            <span className="rounded-md border border-border px-1.5 font-mono text-[11px] text-muted-foreground">
              +{moreHostnames.length}
            </span>
          )}
          {!firstHostname && <span className="text-muted-foreground">None</span>}
        </Fact>
        {repo && (
          <Fact label="Repository">
            <Mono title={repo}>{repo}</Mono>
            {stack.build_source?.branch && (
              <span className="inline-flex items-center gap-1 font-mono text-xs text-muted-foreground">
                <GitBranchIcon className="size-3" aria-hidden />
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
    </header>
  );
};
