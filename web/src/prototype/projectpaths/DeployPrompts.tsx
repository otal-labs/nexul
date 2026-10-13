import { ArrowRight, Rocket } from "lucide-react";
import { Link, useNavigate, useSearchParams } from "react-router";

import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useFetchStacks } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { RepoRole } from "@/enums/Project";
import type { RepoRef } from "@/models/Project";

import { Picker } from "./Picker";

export type DeployPromptVariant = "a" | "b" | "c";

const VARIANTS: { key: DeployPromptVariant; name: string; path: (token: string) => string }[] = [
  { key: "a", name: "Services card", path: (t) => `/projects/${t}/settings/services` },
  { key: "b", name: "Repository row", path: (t) => `/projects/${t}/settings/repositories` },
  { key: "c", name: "Board header", path: (t) => `/board/${t}` },
];

export const useDeployPromptVariant = (): DeployPromptVariant | null => {
  const [params] = useSearchParams();
  const v = params.get("p2");
  return v === "a" || v === "b" || v === "c" ? v : null;
};

// An app repository no stack in the project builds from; empty for anyone who can't read stacks and add services.
export const useWaitingRepos = (projectId: string | undefined): RepoRef[] => {
  const { data: repos } = useFetchProjectRepos(projectId);
  const { data: stacks } = useFetchStacks(projectId);
  const can = useAreaAccess(projectId);
  if (!repos || !stacks || !can?.("stacks") || !can("newProject")) return [];
  return repos.filter(
    (r) =>
      r.role !== RepoRole.Tests &&
      !stacks.some((s) => s.build_source?.repo_owner === r.owner && s.build_source?.repo_name === r.name),
  );
};

export const useDeployPath = (projectId: string) => useWorkspacePath()(`/wizard/project/repository?project=${projectId}&add=1`);

// A: one row per waiting repository inside the Services card, where "No services yet" would be.
export const ServicesWaitingRows = ({ projectId, repos }: { projectId: string; repos: RepoRef[] }) => {
  const to = useDeployPath(projectId);
  return (
    <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
      {repos.map((repo) => (
        <li key={repo.full_name} className="flex items-center gap-3 px-3 py-2.5">
          <span className="min-w-0 flex-1">
            <span className="block truncate font-mono text-xs">{repo.full_name}</span>
            <span className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
              <span aria-hidden className="size-1.5 shrink-0 rounded-full border border-muted-foreground" />
              Attached, not deployed yet
            </span>
          </span>
          <Button asChild variant="outline" size="sm" className="shrink-0">
            <Link to={to}>
              <Rocket className="size-3.5" aria-hidden />
              Deploy this repository
            </Link>
          </Button>
        </li>
      ))}
    </ul>
  );
};

// B: the waiting repository's own row says so and carries the way in.
export const RepoRowDeploy = ({ projectId }: { projectId: string }) => {
  const to = useDeployPath(projectId);
  return (
    <Button asChild variant="outline" size="sm" className="h-8 shrink-0">
      <Link to={to}>
        <Rocket className="size-3.5" aria-hidden />
        Deploy
      </Link>
    </Button>
  );
};

// C: the board header's meta line names the waiting repository, once per project.
export const BoardWaitingMeta = ({ projectId, repos }: { projectId: string; repos: RepoRef[] }) => {
  const to = useDeployPath(projectId);
  const what = repos.length === 1 ? `${repos[0]!.full_name} isn't deployed` : `${repos.length} repositories aren't deployed`;
  return (
    <>
      {" · "}
      <span>{what}</span>{" "}
      <Link to={to} className="group inline-flex items-center gap-0.5 text-foreground underline-offset-4 hover:underline">
        Deploy
        <ArrowRight className="size-3 transition-transform duration-150 ease-standard group-hover:translate-x-0.5" aria-hidden />
      </Link>
    </>
  );
};

export const DeployPromptPicker = ({ token }: { token: string }) => {
  const variant = useDeployPromptVariant();
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  if (!variant) return null;
  const active = VARIANTS.findIndex((v) => v.key === variant);
  return (
    <Picker
      names={VARIANTS.map((v) => v.name)}
      active={active}
      onPick={(i) => navigate(wsPath(`${VARIANTS[i]!.path(token)}?p2=${VARIANTS[i]!.key}`))}
      onReplay={() => undefined}
    />
  );
};
