import { LockIcon } from "lucide-react";
import { useState } from "react";
import { useShallow } from "zustand/react/shallow";

import { RepositoryScanError } from "@/components/wizard/RepositoryScanError";
import { RepositorySearch } from "@/components/wizard/RepositorySearch";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardManualDockerfile } from "@/components/wizard/WizardManualDockerfile";
import { WizardOpenRepositoryRow } from "@/components/wizard/WizardOpenRepositoryRow";
import { useFetchProjectRepos, useSaveTestsAnswer } from "@/hooks/ProjectHooks";
import { useScanRepository } from "@/hooks/RepositoryHooks";
import { useActionState } from "@/hooks/useActionState";
import { useWizardTestsAnswer } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { manualCandidate, preferredCandidate, type Repo } from "@/models/Repository";
import { TestsLocation } from "@/enums/Project";

interface WizardRepositoryStepProps {
  onDone: () => void;
  onBack?: (() => void) | undefined;
}

// Add a service and Attach: picking a repository scans it, then advances, opens the row, or offers a manual Dockerfile.
export const WizardRepositoryStep = ({ onDone, onBack }: WizardRepositoryStepProps) => {
  const scanRepository = useScanRepository();
  const canScan = useActionState({ permission: "projects:create", verb: "scan repositories" });
  const setRepository = useProjectWizardStore((s) => s.setRepository);
  const setScanResult = useProjectWizardStore((s) => s.setScanResult);
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);
  const { projectId, attachStackId, repository, scanResult } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, attachStackId: s.attachStackId, repository: s.repository, scanResult: s.scanResult })),
  );
  const { testsLocation, testsRepo } = useWizardTestsAnswer();
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  const saveTestsAnswer = useSaveTestsAnswer();
  const asksTests = !!projectId && !attachStackId;
  const separateTestsRepo = testsLocation === TestsLocation.Separate ? testsRepo : null;
  // Coming back to the step opens the repository already picked, with its scan.
  const [picked, setPicked] = useState<Repo | null>(repository);

  // The answer is saved on every way out; a failed save is toasted by its hook.
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
      const preferred = preferredCandidate(result);
      if (preferred) setCandidate(preferred);
      if (preferred && !asksTests) await advance();
    } catch {
      // Scan failures render inline below; a failed tests-answer save was already toasted by its hook.
    }
  };

  const scanned = picked && (scanRepository.data ?? (picked.id === repository?.id ? scanResult : null));
  const openRow = scanned && {
    id: picked.id,
    row: (
      <WizardOpenRepositoryRow
        repo={picked}
        scan={scanned}
        onDeploy={scanned.candidates.length > 0 ? () => void advance().catch(() => undefined) : undefined}
      />
    ),
  };

  const continueManually = async (dockerfilePath: string) => {
    if (!picked) return;
    setRepository(picked);
    setCandidate(manualCandidate(picked.name, dockerfilePath));
    await advance().catch(() => undefined);
  };

  return (
    <div>
      <div className="space-y-8">
        <section className="space-y-3">
          {asksTests && <h3 className="text-sm font-medium">Deploys from</h3>}
          {canScan.state === "live" && (
            <RepositorySearch
              onSelect={(repo) => void scan(repo)}
              exclude={separateTestsRepo ? [separateTestsRepo.full_name] : undefined}
              busyId={scanRepository.isPending ? picked?.id : undefined}
              openRow={openRow || undefined}
            />
          )}
          {canScan.state === "locked" && (
            <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <LockIcon className="size-3.5 shrink-0" aria-hidden />
              {canScan.reason}
            </p>
          )}
          {scanRepository.isError && <RepositoryScanError error={scanRepository.error} />}
          {!!scanned && scanned.candidates.length === 0 && <WizardManualDockerfile onContinue={(path) => void continueManually(path)} />}
        </section>
        {asksTests && <TestsLocationChoice />}
      </div>
      <WizardFooter onBack={onBack} />
    </div>
  );
};
