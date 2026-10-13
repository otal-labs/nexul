import { ArrowRight, Check } from "lucide-react";

import { RepoProviderMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import type { Repo } from "@/models/Repository";

import { PICKED, SCAN_LINE } from "./mock";
import { SectionLabel, StepHeading, type RepoStepProps } from "./parts";
import { SearchMock } from "./SearchMock";

interface PickedRowProps {
  repo: Repo;
  onDeploy: () => void;
  onAttach: () => void;
}

// The picked row opens in place: what the scan found, then the two ways on, deploying first.
const PickedRow = ({ repo, onDeploy, onAttach }: PickedRowProps) => (
  <li className="relative bg-accent px-2 py-3" aria-current="true">
    <span aria-hidden className="absolute inset-y-0 left-0 w-0.5 bg-brand" />
    <div className="flex items-center gap-3">
      <RepoProviderMark provider={repo.provider} />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium">{repo.full_name}</span>
        <span className="flex min-w-0 items-center gap-1.5 font-mono text-xs text-muted-foreground">
          <span className="shrink-0">{repo.default_branch}</span>
          <span aria-hidden>·</span>
          <Check className="size-3 shrink-0 text-success" strokeWidth={3} aria-hidden />
          <span className="truncate">{SCAN_LINE}</span>
        </span>
      </span>
    </div>
    <div className="mt-3 flex flex-wrap items-center justify-end gap-2">
      <Button type="button" variant="ghost" size="sm" onClick={onAttach}>
        Attach without deploying
      </Button>
      <Button type="button" size="sm" onClick={onDeploy}>
        Deploy it
      </Button>
    </div>
  </li>
);

// Variant 2: picking opens the row with Deploy it and Attach without deploying; No repository yet is a quiet line under the list.
export const RepoStepRow = ({ picked, onPick, onExit }: RepoStepProps) => (
  <StepHeading description="Pick the repository this project's code lives in. Nexul reads its Dockerfile or compose file to deploy it.">
    <div className="mt-4 space-y-4">
      <TestsLocationChoice />
      <SectionLabel>Repository</SectionLabel>
      <SearchMock
        onPick={onPick}
        renderRow={(repo) =>
          picked &&
          repo.id === PICKED.id && <PickedRow key={repo.id} repo={repo} onDeploy={() => undefined} onAttach={() => onExit("attached")} />
        }
        after={
          <p className="flex flex-wrap items-center gap-x-2 text-sm text-muted-foreground">
            <span>No code to point at?</span>
            <button
              type="button"
              onClick={() => onExit("none")}
              className="group inline-flex items-center gap-1 font-medium text-foreground underline-offset-4 outline-none hover:underline focus-visible:underline"
            >
              No repository yet
              <ArrowRight className="size-3.5 transition-transform duration-150 ease-standard group-hover:translate-x-0.5" aria-hidden />
            </button>
          </p>
        }
      />
      <WizardFooter onBack={() => undefined} />
    </div>
  </StepHeading>
);
