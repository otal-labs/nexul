import { AlertCircle, ExternalLink } from "lucide-react";
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
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useFetchProjectRepos, useSaveTestsAnswer } from "@/hooks/ProjectHooks";
import { useScanRepository } from "@/hooks/RepositoryHooks";
import { manualCandidate, parseInstallUrl, type Repo } from "@/models/Repository";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { TestsLocation } from "@/enums/Project";

interface WizardRepositoryStepProps {
  onDone: () => void;
  onSkip?: (() => void) | undefined;
}

// Searches installation repositories (spec §5/§7); picking one scans it. A scan can land in three places: candidates
// found (advance), nothing found (offer a manual Dockerfile candidate), or the App isn't installed (link to fix it).
export const WizardRepositoryStep = ({ onDone, onSkip }: WizardRepositoryStepProps) => {
  const scanRepository = useScanRepository();
  const setRepository = useProjectWizardStore((s) => s.setRepository);
  const setScanResult = useProjectWizardStore((s) => s.setScanResult);
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);
  const { projectId, attachStackId, testsLocation, testsRepo } = useProjectWizardStore(
    useShallow((s) => ({
      projectId: s.projectId,
      attachStackId: s.attachStackId,
      testsLocation: s.testsLocation,
      testsRepo: s.testsRepo,
    })),
  );
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  const saveTestsAnswer = useSaveTestsAnswer();
  const asksTests = !!projectId && !attachStackId;
  const separateTestsRepo = testsLocation === TestsLocation.Separate ? testsRepo : null;
  const [picked, setPicked] = useState<Repo | null>(null);
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");

  // The answer is saved on the way out so it lands whichever way the step is left; a failed save is toasted.
  const advance = async () => {
    if (asksTests && testsLocation) {
      await saveTestsAnswer.mutateAsync({ projectId, testsLocation, testsRepo, attached: projectRepos ?? [] });
    }
    onDone();
  };

  const scan = async (repo: Repo) => {
    setPicked(repo);
    try {
      const result = await scanRepository.mutateAsync({ owner: repo.owner, name: repo.name });
      setRepository(repo);
      setScanResult(result);
      if (result.candidates.length > 0) {
        const preferred = result.candidates.find((c) => c.kind === "compose") ?? result.candidates[0]!;
        setCandidate(preferred);
        await advance();
      }
    } catch {
      // Scan failures render inline below; a failed tests-answer save was already toasted by its hook.
    }
  };

  const continueManually = async () => {
    if (!picked || !dockerfilePath.trim()) return;
    setRepository(picked);
    setCandidate(manualCandidate(picked.name, dockerfilePath.trim()));
    // A failed tests-answer save was already toasted by its hook; the step stays put.
    await advance().catch(() => undefined);
  };

  const notInstalled =
    scanRepository.isError &&
    (scanRepository.error as { response?: { status?: number } })?.response?.status === 404;
  const installUrl = notInstalled ? parseInstallUrl(errorMessage(scanRepository.error)) : undefined;
  const nothingFound = scanRepository.isSuccess && scanRepository.data.candidates.length === 0;

  return (
    <div className="space-y-4">
      {asksTests && <TestsLocationChoice />}
      {asksTests && <p className="text-sm font-medium">Repository to deploy</p>}
      <RepositorySearch
        onSelect={(repo) => void scan(repo)}
        excludeId={separateTestsRepo?.id}
        busyId={scanRepository.isPending ? picked?.id : undefined}
      />
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
      <WizardFooter skip={onSkip && <WizardSkipLink onClick={onSkip} />} />
    </div>
  );
};
