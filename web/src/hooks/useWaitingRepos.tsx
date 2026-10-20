import { useEffect } from "react";
import { useParams, useSearchParams } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useScanRepository } from "@/hooks/RepositoryHooks";
import { useFetchServices } from "@/hooks/ServiceHooks";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { useWizardStackId } from "@/hooks/useWizardSetup";
import { useWizardDoor } from "@/hooks/useWizardStepOrder";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { preferredCandidate } from "@/models/Repository";
import type { RepoRef } from "@/models/Project";
import { RepoRole } from "@/enums/Project";

// A project's app repositories no service builds from; undefined while they load, and without stacks read, where
// every one would look waiting. Whether the viewer can deploy one is the row's own locked or hidden door.
export const useWaitingReposState = (projectId: string | undefined): { waiting: RepoRef[] | undefined; pending: boolean } => {
  const can = useAreaAccess(projectId);
  const allowed = !!projectId && !!can && can("stacks") && can("projects");
  const { data: repos } = useFetchProjectRepos(allowed ? projectId : undefined);
  const { data: services } = useFetchServices(projectId, allowed);
  if (!allowed || !repos || !services) return { waiting: undefined, pending: allowed };
  const waiting = repos.filter(
    (r) =>
      r.role !== RepoRole.Tests &&
      !services.some((s) => s.build_source?.repo_owner === r.owner && s.build_source?.repo_name === r.name),
  );
  return { waiting, pending: false };
};

export const useWaitingRepos = (projectId: string | undefined): RepoRef[] | undefined => useWaitingReposState(projectId).waiting;

export interface ScanTarget {
  owner: string;
  name: string;
  fullName: string;
  provider: string;
}

const targetOf = (r: RepoRef): ScanTarget => ({ owner: r.owner, name: r.name, fullName: r.full_name, provider: r.connector_id });

// What Deploy and Service read: the repository the run picked, else the one ?repo= names, else the first waiting.
export const useScanTarget = (): { target: ScanTarget | undefined; pending: boolean } => {
  const { step } = useParams();
  const door = useWizardDoor();
  const [searchParams] = useSearchParams();
  const projectId = useProjectWizardStore((s) => s.projectId);
  const picked = useProjectWizardStore((s) => s.repository);
  const { waiting, pending } = useWaitingReposState(door === "attach" ? undefined : (projectId ?? undefined));
  const named = searchParams.get("repo");
  const pickedTarget = picked ? { owner: picked.owner, name: picked.name, fullName: picked.full_name, provider: picked.provider } : undefined;
  const byName = waiting?.find((r) => r.full_name === named);
  const only = waiting?.length === 1 ? waiting[0] : undefined;

  if (door === "attach") return { target: pickedTarget, pending: false };
  if (step === "deploy" || step === "service") {
    const chosen = byName ?? waiting?.[0];
    return { target: pickedTarget ?? (chosen && targetOf(chosen)), pending: !pickedTarget && pending };
  }
  const linked = door === "add" && step === "repository" && !picked ? (named ? byName : only) : undefined;
  return { target: linked && targetOf(linked), pending: false };
};

// Scans the target once so Deploy and Service open on it; a failed read stays failed until Repository resets it.
export const useSeedScan = () => {
  const { step } = useParams();
  const door = useWizardDoor();
  const goTo = useWizardGoTo();
  const stackId = useWizardStackId();
  const candidate = useProjectWizardStore((s) => s.candidate);
  const { target } = useScanTarget();
  const { mutateAsync: scan } = useScanRepository();
  const { owner, name, fullName, provider } = target ?? {};

  useEffect(() => {
    if (!owner || !name || !fullName || !provider || candidate || stackId) return;
    const store = useProjectWizardStore.getState();
    if (store.seed.status !== "idle" || store.scanResult) return;
    store.setSeed({ status: "running", repository: fullName, error: null });
    const seed = async () => {
      try {
        const result = await scan({ owner, name });
        const held = useProjectWizardStore.getState();
        if (held.seed.repository !== fullName) return;
        if (held.repository?.full_name !== fullName) {
          held.setRepository({ id: 0, owner, name, full_name: fullName, default_branch: result.default_branch, html_url: "", provider });
        }
        const found = preferredCandidate(result);
        held.setScanResult(result);
        if (found) held.setCandidate(found);
        held.clearSeed();
        if (found && door === "add" && step === "repository") goTo("service");
      } catch (error) {
        const held = useProjectWizardStore.getState();
        if (held.seed.repository === fullName) held.setSeed({ status: "failed", repository: fullName, error });
      }
    };
    void seed();
  }, [step, door, owner, name, fullName, provider, candidate, stackId, scan, goTo]);
};
