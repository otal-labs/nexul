import type { ReactNode } from "react";
import { GitBranchIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import type { Stack } from "@/models/Stack";
import { cn } from "@/lib/utils";

interface StackMetaLineProps {
  stack: Stack;
  image: string | undefined;
}

// Each fact carries the dot before it in a 1.25rem slot; the row is pulled 1.25rem left inside a clipping box, so the dot of
// whichever fact starts a line falls outside and no line starts or ends on a dot. A long fact shortens on its own, and a
// copy or the tooltip still gives the whole value.
const Fact = ({ title, value, wide = false, children }: { title: string; value: string; wide?: boolean; children: ReactNode }) => (
  <span className={cn("inline-flex min-w-0 items-center", wide && "max-w-full flex-[1_1_auto]")}>
    <span aria-hidden className="w-5 shrink-0 text-center text-muted-foreground/50">·</span>
    {/* A whole number of mono characters wide, with slack for rounding, so a cut fact's ellipsis sits flush. */}
    <span className={cn("inline-flex min-w-0 items-center gap-1 font-mono text-xs", !wide && "max-w-[calc(36ch+0.5px)]")} title={`${title}: ${value}`}>
      {children}
    </span>
  </span>
);

const repoLabel = (stack: Stack): string | undefined => {
  const src = stack.build_source;
  if (!src?.repo_owner || !src.repo_name) return undefined;
  return `${src.repo_owner}/${src.repo_name}`;
};

const fileName = (path: string): string => path.slice(path.lastIndexOf("/") + 1);

// What the stack is, under its title: the slug, then where it runs and what it is built from.
export const StackMetaLine = ({ stack, image }: StackMetaLineProps) => {
  const repo = repoLabel(stack);
  const branch = stack.build_source?.branch;
  return (
    <span className="block min-w-0 overflow-hidden">
    <span className="-ml-5 flex min-w-0 flex-wrap items-center gap-y-0.5 text-muted-foreground">
      <Fact title="Slug" value={stack.slug}>
        <span className="truncate">{stack.slug}</span>
      </Fact>
      <Fact title="Runner" value={stack.machine}>
        <span className="truncate">{stack.machine}</span>
      </Fact>
      <Fact title="Strategy" value={stack.compose_path ? `${stack.strategy} ${stack.compose_path}` : stack.strategy}>
        <span className="truncate">{stack.compose_path ? `${stack.strategy} ${fileName(stack.compose_path)}` : stack.strategy}</span>
      </Fact>
      {repo && (
        <Fact wide title="Repository" value={branch ? `${repo} ${branch}` : repo}>
          <span className="truncate">{repo}</span>
          {branch && <GitBranchIcon className="size-3 shrink-0" aria-hidden />}
          {branch && <span className="truncate">{branch}</span>}
        </Fact>
      )}
      {!repo && stack.docker_network && (
        <Fact title="Network" value={stack.docker_network}>
          <span className="truncate">{stack.docker_network}</span>
        </Fact>
      )}
      {stack.strategy !== "compose" && image && (
        <Fact wide title="Image" value={image}>
          <span className="truncate">{image}</span>
        </Fact>
      )}
      {!stack.managed && <NoFillBadge color="bg-muted-foreground" className="ml-5">unmanaged</NoFillBadge>}
    </span>
    </span>
  );
};
