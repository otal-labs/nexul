import type { ReactNode } from "react";
import { GitBranchIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import type { Stack } from "@/models/Stack";

interface StackMetaLineProps {
  stack: Stack;
  image: string | undefined;
}

// Image refs, paths and names here are identifiers someone copies, so they wrap instead of truncating.
const Mono = ({ title, children }: { title: string; children: ReactNode }) => (
  <span className="inline-flex min-w-0 items-center gap-1 font-mono text-xs wrap-anywhere" title={title}>
    {children}
  </span>
);

const Dot = () => <span aria-hidden className="text-muted-foreground/50">·</span>;

const repoLabel = (stack: Stack): string | undefined => {
  const src = stack.build_source;
  if (!src?.repo_owner || !src.repo_name) return undefined;
  return `${src.repo_owner}/${src.repo_name}`;
};

// What the stack is, under its title: the slug, then where it runs and what it is built from.
export const StackMetaLine = ({ stack, image }: StackMetaLineProps) => {
  const repo = repoLabel(stack);
  const strategy = stack.compose_path ? `${stack.strategy} ${stack.compose_path}` : stack.strategy;
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-muted-foreground">
      <Mono title="Slug">{stack.slug}</Mono>
      <Dot />
      <Mono title="Runner">{stack.machine}</Mono>
      <Dot />
      <Mono title="Strategy">{strategy}</Mono>
      {repo && (
        <>
          <Dot />
          <Mono title="Repository">
            {repo}
            {stack.build_source?.branch && (
              <>
                <GitBranchIcon className="size-3 shrink-0" aria-hidden />
                {stack.build_source.branch}
              </>
            )}
          </Mono>
        </>
      )}
      {!repo && stack.docker_network && (
        <>
          <Dot />
          <Mono title="Network">{stack.docker_network}</Mono>
        </>
      )}
      {stack.strategy !== "compose" && image && (
        <>
          <Dot />
          <Mono title="Image">{image}</Mono>
        </>
      )}
      {!stack.managed && <NoFillBadge color="bg-muted-foreground">unmanaged</NoFillBadge>}
    </span>
  );
};
