import { useState } from "react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { RepositorySearch } from "@/components/wizard/RepositorySearch";
import { WizardRepositoryCard } from "@/components/wizard/WizardRepositoryCard";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useWizardTestsAnswer } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { RepoRole, TestsLocation } from "@/enums/Project";

const TestsRepoPicker = () => {
  const { projectId, repository, setTestsRepo } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, repository: s.repository, setTestsRepo: s.setTestsRepo })),
  );
  const { testsRepo } = useWizardTestsAnswer();
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  const [changing, setChanging] = useState(false);
  // A repository is attached once, in one role, so the one this project deploys can't double as its tests.
  const deployed = (projectRepos ?? []).filter((r) => r.role === RepoRole.App).map((r) => r.full_name);
  const exclude = repository ? [...deployed, repository.full_name] : deployed;

  return (
    <div className="space-y-3">
      {testsRepo && (
        <WizardRepositoryCard
          fullName={testsRepo.full_name}
          provider={testsRepo.provider}
          meta="Agents read and run it. Never deployed."
          action={
            <Button type="button" variant="outline" size="sm" onClick={() => setChanging(!changing)}>
              {changing ? "Cancel" : "Change"}
            </Button>
          }
        />
      )}
      {(!testsRepo || changing) && (
        <RepositorySearch
          label="Search tests repositories"
          onSelect={(repo) => {
            setTestsRepo(repo);
            setChanging(false);
          }}
          exclude={exclude}
        />
      )}
    </div>
  );
};

export const TestsLocationChoice = () => {
  const setTestsLocation = useProjectWizardStore((s) => s.setTestsLocation);
  const { testsLocation } = useWizardTestsAnswer();

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h3 id="tests-location" className="text-sm font-medium">
          Tests live in
        </h3>
        <ToggleGroup
          type="single"
          variant="segmented"
          size="xs"
          aria-labelledby="tests-location"
          value={testsLocation ?? ""}
          onValueChange={(value) => value && setTestsLocation(value as TestsLocation)}
        >
          <ToggleGroupItem value={TestsLocation.Same}>The same repository</ToggleGroupItem>
          <ToggleGroupItem value={TestsLocation.Separate}>A separate one</ToggleGroupItem>
        </ToggleGroup>
      </div>
      {testsLocation === TestsLocation.Separate && <TestsRepoPicker />}
    </section>
  );
};
