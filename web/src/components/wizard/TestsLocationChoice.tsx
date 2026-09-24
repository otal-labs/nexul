import { useShallow } from "zustand/react/shallow";

import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchRepositories } from "@/hooks/RepositoryHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { TestsLocation } from "@/enums/Project";

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
  const { data: repos } = useFetchRepositories();
  const { testsRepo, setTestsRepo } = useProjectWizardStore(
    useShallow((s) => ({ testsRepo: s.testsRepo, setTestsRepo: s.setTestsRepo })),
  );

  return (
    <div className="space-y-2 pt-3">
      <p className="text-xs text-muted-foreground">Attached to the project for the agent to read and run. Never deployed.</p>
      <Select
        value={testsRepo ? String(testsRepo.id) : ""}
        onValueChange={(id) => {
          const repo = repos?.find((r) => String(r.id) === id);
          if (repo) setTestsRepo(repo);
        }}
      >
        <SelectTrigger aria-label="Tests repository" className="w-full">
          <SelectValue placeholder="Choose the tests repository…" />
        </SelectTrigger>
        <SelectContent>
          {repos?.map((repo) => (
            <SelectItem key={repo.id} value={String(repo.id)}>
              {repo.full_name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};

// Asked before the repository to deploy is picked, because picking it moves the wizard on.
export const TestsLocationChoice = () => {
  const { testsLocation, setTestsLocation } = useProjectWizardStore(
    useShallow((s) => ({ testsLocation: s.testsLocation, setTestsLocation: s.setTestsLocation })),
  );

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
