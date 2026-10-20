import { LockIcon } from "lucide-react";
import { useState } from "react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { RepositorySearch } from "@/components/wizard/RepositorySearch";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardRepositoryCard } from "@/components/wizard/WizardRepositoryCard";
import { useAddProjectRepo, useFetchProjectRepos, useSaveTestsAnswer } from "@/hooks/ProjectHooks";
import { useActionState } from "@/hooks/useActionState";
import { useWizardStackId, useWizardTestsAnswer } from "@/hooks/useWizardSetup";
import type { SetupMark } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { RepoRole, TestsLocation } from "@/enums/Project";

interface WizardSetupRepositoryStepProps {
  onDone: () => void;
  onBack: () => void;
  onLeaveEarly: (repository: SetupMark) => Promise<void>;
}

// Setup's Repository step only names the repository and where the tests live; Deploy reads it next.
export const WizardSetupRepositoryStep = ({ onDone, onBack, onLeaveEarly }: WizardSetupRepositoryStepProps) => {
  const canScan = useActionState({ permission: "projects:create", verb: "scan repositories" });
  const { projectId, repository, setRepository, clearSeed } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, repository: s.repository, setRepository: s.setRepository, clearSeed: s.clearSeed })),
  );
  const { testsLocation, testsRepo } = useWizardTestsAnswer();
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  const attachedApp = projectRepos?.find((r) => r.role === RepoRole.App);
  const saveTestsAnswer = useSaveTestsAnswer();
  const addRepo = useAddProjectRepo();
  const stackId = useWizardStackId();
  const [changing, setChanging] = useState(false);
  const separateTestsRepo = testsLocation === TestsLocation.Separate ? testsRepo : null;
  const attached = (owner: string, name: string) => !!projectRepos?.some((r) => r.owner === owner && r.name === name);
  const held = repository
    ? { fullName: repository.full_name, provider: repository.provider, attached: attached(repository.owner, repository.name) }
    : attachedApp && { fullName: attachedApp.full_name, provider: attachedApp.connector_id, attached: true };

  // The answer is saved on every way out; a failed save is toasted by its hook, and the step stays for another try.
  const saveTests = async () => {
    if (projectId && testsLocation) {
      await saveTestsAnswer.mutateAsync({ projectId, testsLocation, testsRepo, attached: projectRepos ?? [] });
    }
  };
  const goOn = async () => {
    await saveTests();
    if (projectId && repository && !attached(repository.owner, repository.name)) {
      await addRepo.mutateAsync({ projectId, owner: repository.owner, name: repository.name, connectorId: repository.provider });
    }
    clearSeed();
    onDone();
  };
  const withoutRepository = async () => {
    await saveTests();
    await onLeaveEarly("skipped");
  };

  return (
    <div>
      <div className="space-y-8">
        <section className="space-y-3">
          <h3 className="text-sm font-medium">Deploys from</h3>
          {held && (
            <WizardRepositoryCard
              fullName={held.fullName}
              provider={held.provider}
              meta={held.attached ? "Attached to this project" : "Attaches when you continue"}
              action={
                <Button type="button" variant="outline" size="sm" onClick={() => setChanging(!changing)}>
                  {changing ? "Cancel" : "Change"}
                </Button>
              }
            />
          )}
          {(!held || changing) && canScan.state === "live" && (
            <RepositorySearch
              onSelect={(repo) => {
                setRepository(repo);
                setChanging(false);
              }}
              exclude={separateTestsRepo ? [separateTestsRepo.full_name] : undefined}
            />
          )}
          {(!held || changing) && canScan.state === "locked" && (
            <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <LockIcon className="size-3.5 shrink-0" aria-hidden />
              {canScan.reason}
            </p>
          )}
        </section>
        <TestsLocationChoice />
      </div>
      <WizardFooter
        onBack={onBack}
        skip={
          !held &&
          !stackId && (
            <Button type="button" variant="ghost" className="text-muted-foreground" onClick={() => void withoutRepository().catch(() => undefined)}>
              No repository yet
            </Button>
          )
        }
      >
        {held && !changing && (
          <Button type="button" onClick={() => void goOn().catch(() => undefined)} loading={saveTestsAnswer.isPending || addRepo.isPending}>
            Continue
          </Button>
        )}
      </WizardFooter>
    </div>
  );
};
