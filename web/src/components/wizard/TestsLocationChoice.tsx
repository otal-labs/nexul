import { useShallow } from "zustand/react/shallow";

import { RepoProviderMark } from "@/components/ProviderMarks";
import { RepositorySearch } from "@/components/wizard/RepositorySearch";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useFetchProjectRepos } from "@/hooks/ProjectHooks";
import { useWizardTestsAnswer } from "@/hooks/useWizardSetup";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { RepoRole, TestsLocation } from "@/enums/Project";

interface TestsLocationRowProps {
  value: TestsLocation;
  label: string;
  description: string;
}

const TestsLocationRow = ({ value, label, description }: TestsLocationRowProps) => (
  <label
    htmlFor={`tests-location-${value}`}
    className="-mx-2 flex cursor-pointer items-start gap-3 rounded-md px-2 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40"
  >
    <RadioGroupItem id={`tests-location-${value}`} value={value} className="mt-0.5" />
    <span className="min-w-0">
      <span className="block text-sm font-medium">{label}</span>
      <span className="mt-0.5 block text-sm text-muted-foreground">{description}</span>
    </span>
  </label>
);

const TestsRepoPicker = () => {
  const { projectId, repository, setTestsRepo } = useProjectWizardStore(
    useShallow((s) => ({ projectId: s.projectId, repository: s.repository, setTestsRepo: s.setTestsRepo })),
  );
  const { testsRepo } = useWizardTestsAnswer();
  const { data: projectRepos } = useFetchProjectRepos(projectId ?? undefined);
  // A repository is attached once, in one role, so the one this project deploys can't double as its tests.
  const deployed = (projectRepos ?? []).filter((r) => r.role === RepoRole.App).map((r) => r.full_name);
  const exclude = repository ? [...deployed, repository.full_name] : deployed;

  return (
    <div className="space-y-2 pt-3">
      <p className="text-xs text-muted-foreground">Attached to the project for the agent to read and run. Never deployed.</p>
      {testsRepo && (
        <p className="flex items-center gap-2 text-sm">
          <RepoProviderMark provider={testsRepo.provider} />
          <span className="min-w-0 truncate font-medium">{testsRepo.full_name}</span>
        </p>
      )}
      <RepositorySearch label="Search tests repositories" onSelect={setTestsRepo} exclude={exclude} />
    </div>
  );
};

// Asked before the repository to deploy is picked, because picking it moves the wizard on.
export const TestsLocationChoice = () => {
  const setTestsLocation = useProjectWizardStore((s) => s.setTestsLocation);
  const { testsLocation } = useWizardTestsAnswer();

  return (
    <div>
      <p className="text-sm font-medium">Where do this project's tests live?</p>
      <RadioGroup
        aria-label="Where tests live"
        value={testsLocation ?? ""}
        onValueChange={(value) => setTestsLocation(value as TestsLocation)}
        className="mt-2 gap-0 divide-y divide-border border-y border-border"
      >
        <TestsLocationRow
          value={TestsLocation.Same}
          label="In the repository you deploy"
          description="Tests sit next to the code they cover."
        />
        <TestsLocationRow
          value={TestsLocation.Separate}
          label="In a separate repository"
          description="A tests repository, attached to the project but never deployed."
        />
      </RadioGroup>
      {testsLocation === TestsLocation.Separate && <TestsRepoPicker />}
    </div>
  );
};
