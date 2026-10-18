import { AlertCircle, ExternalLink, LockIcon } from "lucide-react";
import { useState } from "react";
import { useShallow } from "zustand/react/shallow";

import { errorMessage } from "@/api/client";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RepositorySearch } from "@/components/wizard/RepositorySearch";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardNoRepositoryLine } from "@/components/wizard/WizardNoRepositoryLine";
import { WizardOpenRepositoryRow } from "@/components/wizard/WizardOpenRepositoryRow";
import { useAddProjectRepo, useFetchProjectRepos, useSaveTestsAnswer } from "@/hooks/ProjectHooks";
import { useScanRepository } from "@/hooks/RepositoryHooks";
import { useActionState } from "@/hooks/useActionState";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import type { SetupMark } from "@/models/Project";
import { manualCandidate, parseInstallUrl, preferredCandidate, type Repo } from "@/models/Repository";
import { TestsLocation } from "@/enums/Project";

interface WizardRepositoryStepProps {
  onDone: () => void;
  onBack?: (() => void) | undefined;
  // Offered while the project is set up: ends setup's deploy path here, with the repository attached or none.
  onLeaveEarly?: ((repository: SetupMark) => Promise<void>) | undefined;
}

// Searches installation repositories (spec §5/§7); picking one scans it. A scan can land in three places: candidates
// found (advance, or open the row while setting up), nothing found (a manual Dockerfile), or the App isn't installed.
export const WizardRepositoryStep = ({ onDone, onBack, onLeaveEarly }: WizardRepositoryStepProps) => {
  const scanRepository = useScanRepository();
  const canScan = useActionState({ permission: "projects:create", verb: "scan repositories" });
  const setRepository = useProjectWizardStore((s) => s.setRepository);
  const setScanResult = useProjectWizardStore((s) => s.setScanResult);
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);
  const { projectId, attachStackId, testsLocation, testsRepo, repository, scanResult } = useProjectWizardStore(
    useShallow((s) => ({
      projectId: s.projectId,
      attachStackId: s.attachStackId,
      testsLocation: s.testsLocation,
      testsRepo: s.testsRepo,
      repository: s.repository,
      scanResult: s.scanResult,
    })),
  );
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  const saveTestsAnswer = useSaveTestsAnswer();
  const addRepo = useAddProjectRepo();
  const asksTests = !!projectId && !attachStackId;
  const separateTestsRepo = testsLocation === TestsLocation.Separate ? testsRepo : null;
  // Coming back to the step opens the repository already picked, with its scan.
  const [picked, setPicked] = useState<Repo | null>(repository);
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");

  // The answer is saved on every way out, so it lands whichever way the step is left; a failed save is toasted.
  const saveTests = async () => {
    if (asksTests && testsLocation) {
      await saveTestsAnswer.mutateAsync({ projectId, testsLocation, testsRepo, attached: projectRepos ?? [] });
    }
  };
  const advance = async () => {
    await saveTests();
    onDone();
  };

  const scan = async (repo: Repo) => {
    setPicked(repo);
    try {
      const result = await scanRepository.mutateAsync({ owner: repo.owner, name: repo.name });
      setRepository(repo);
      setScanResult(result);
      const preferred = preferredCandidate(result);
      if (preferred) setCandidate(preferred);
      if (preferred && !onLeaveEarly) await advance();
    } catch {
      // Scan failures render inline below; a failed tests-answer save was already toasted by its hook.
    }
  };

  // Failures are toasted by their hooks; the step stays for another try.
  const leaveWithoutRepository = async () => {
    await saveTests();
    await onLeaveEarly?.("skipped");
  };
  const attachWithoutDeploying = async () => {
    if (!picked || !projectId) return;
    const attached = projectRepos?.some((r) => r.owner === picked.owner && r.name === picked.name);
    await saveTests();
    if (!attached) await addRepo.mutateAsync({ projectId, owner: picked.owner, name: picked.name, connectorId: picked.provider });
    await onLeaveEarly?.("done");
  };

  const scanned = picked && (scanRepository.data ?? (picked.id === repository?.id ? scanResult : null));
  const openRow = scanned && {
    id: picked.id,
    row: (
      <WizardOpenRepositoryRow
        repo={picked}
        scan={scanned}
        attaching={addRepo.isPending || saveTestsAnswer.isPending}
        onAttach={onLeaveEarly && (() => void attachWithoutDeploying().catch(() => undefined))}
        onDeploy={scanned.candidates.length > 0 ? () => void advance().catch(() => undefined) : undefined}
      />
    ),
  };

  const continueManually = async () => {
    if (!picked || !dockerfilePath.trim()) return;
    setRepository(picked);
    setCandidate(manualCandidate(picked.name, dockerfilePath.trim()));
    // A failed tests-answer save was already toasted by its hook; the step stays put.
    await advance().catch(() => undefined);
  };

  const noRepositoryLine = onLeaveEarly && <WizardNoRepositoryLine onClick={() => void leaveWithoutRepository().catch(() => undefined)} />;

  const notInstalled =
    scanRepository.isError &&
    (scanRepository.error as { response?: { status?: number } })?.response?.status === 404;
  const installUrl = notInstalled ? parseInstallUrl(errorMessage(scanRepository.error)) : undefined;
  const nothingFound = !!scanned && scanned.candidates.length === 0;

  return (
    <div className="space-y-4">
      {asksTests && <TestsLocationChoice />}
      {asksTests && <p className="text-sm font-medium">Repository</p>}
      {canScan.state === "live" && (
        <RepositorySearch
          onSelect={(repo) => void scan(repo)}
          excludeId={separateTestsRepo?.id}
          busyId={scanRepository.isPending ? picked?.id : undefined}
          openRow={openRow || undefined}
          after={noRepositoryLine}
        />
      )}
      {canScan.state === "locked" && (
        <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <LockIcon className="size-3.5 shrink-0" aria-hidden />
          {canScan.reason}
        </p>
      )}
      {canScan.state !== "live" && noRepositoryLine}
      {notInstalled && (
        <EmptyState
          icon={AlertCircle}
          title="The GitHub App isn't installed on this repository"
          message={errorMessage(scanRepository.error)}
          size="compact"
          action={
            installUrl && (
              <Button asChild variant="outline" size="sm">
                <a href={installUrl} target="_blank" rel="noreferrer">
                  Install the GitHub App
                  <ExternalLink className="size-3.5" aria-hidden />
                </a>
              </Button>
            )
          }
        />
      )}
      {scanRepository.isError && !notInstalled && <ErrorDisplay error={scanRepository.error} title="Couldn't scan the repository." />}
      {nothingFound && (
        <EmptyState
          title="Nothing to deploy found"
          message="No Dockerfile or compose file found. Enter a Dockerfile path to continue."
          size="compact"
          action={
            <div className="flex flex-wrap items-center justify-center gap-2">
              <Input
                value={dockerfilePath}
                onChange={(e) => setDockerfilePath(e.target.value)}
                placeholder="Dockerfile"
                aria-label="Dockerfile path"
                className="h-8 w-40"
              />
              <Button size="sm" onClick={() => void continueManually()} disabled={!dockerfilePath.trim()}>
                Continue manually
              </Button>
            </div>
          }
        />
      )}
      <WizardFooter onBack={onBack} />
    </div>
  );
};
