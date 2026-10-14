import { useEffect, useRef } from "react";
import { useParams, useSearchParams } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useScanRepository } from "@/hooks/RepositoryHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import type { RepoRef } from "@/models/Project";
import { RepoRole } from "@/enums/Project";

// A project's app repositories no service builds from; undefined without stacks read, where every one would look waiting.
// Whether the viewer can deploy one is the row's own locked or hidden door.
export const useWaitingRepos = (projectId: string | undefined): RepoRef[] | undefined => {
  const can = useAreaAccess(projectId);
  const allowed = !!projectId && !!can && can("stacks") && can("projects");
  const { data: repos } = useFetchProjectRepos(allowed ? projectId : undefined);
  const { data: services } = useFetchServices(projectId, allowed);
  if (!allowed || !repos || !services) return undefined;
  return repos.filter(
    (r) =>
      r.role !== RepoRole.Tests &&
      !services.some((s) => s.build_source?.repo_owner === r.owner && s.build_source?.repo_name === r.name),
  );
};

// Add a service skips the repository pick when one waits to deploy: the one ?repo= names, else the only one.
export const useSeedWaitingRepository = () => {
  const { step } = useParams();
  const [searchParams] = useSearchParams();
  const projectId = useProjectWizardStore((s) => s.projectId);
  const picked = useProjectWizardStore((s) => !!s.repository);
  const waiting = useWaitingRepos(searchParams.has("add") ? (projectId ?? undefined) : undefined);
  const named = searchParams.get("repo");
  const only = waiting?.length === 1 ? waiting[0] : undefined;
  const target = named ? waiting?.find((r) => r.full_name === named) : only;
  const { mutateAsync: scan } = useScanRepository();
  const goTo = useWizardGoTo();
  const seeded = useRef(false);

  useEffect(() => {
    if (step !== "repository" || !target || picked || seeded.current) return;
    seeded.current = true;
    const seed = async () => {
      const result = await scan({ owner: target.owner, name: target.name });
      const candidate = result.candidates.find((c) => c.kind === "compose") ?? result.candidates[0];
      const store = useProjectWizardStore.getState();
      store.setRepository({
        id: 0,
        owner: target.owner,
        name: target.name,
        full_name: target.full_name,
        default_branch: result.default_branch,
        html_url: "",
        provider: target.connector_id,
      });
      store.setScanResult(result);
      if (!candidate) return;
      store.setCandidate(candidate);
      goTo("service");
    };
    // A failed scan leaves the step's own search to pick and scan it again.
    void seed().catch(() => undefined);
  }, [step, target, picked, scan, goTo]);
};
