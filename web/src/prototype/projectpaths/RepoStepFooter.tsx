import { Button } from "@/components/ui/button";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";

import { PickedSummary, SectionLabel, StepHeading, type RepoStepProps } from "./parts";
import { SearchMock } from "./SearchMock";

// Variant 1: the choices live in the footer where Skip for now was; the picked repository waits for a choice.
export const RepoStepFooter = ({ picked, onPick, onUnpick, onExit }: RepoStepProps) => (
  <StepHeading description="Pick the repository this project's code lives in. Nexul reads its Dockerfile or compose file to deploy it.">
    <div className="mt-4 space-y-4">
      <TestsLocationChoice />
      <SectionLabel>Repository</SectionLabel>
      {!picked && <SearchMock onPick={onPick} />}
      {picked && <PickedSummary onChange={onUnpick} />}
      {!picked && (
        <WizardFooter
          onBack={() => undefined}
          skip={
            <Button type="button" variant="ghost" className="text-muted-foreground" onClick={() => onExit("none")}>
              No repository yet
            </Button>
          }
        />
      )}
      {picked && (
        <WizardFooter
          onBack={onUnpick}
          skip={
            <Button type="button" variant="ghost" className="text-muted-foreground" onClick={() => onExit("attached")}>
              Attach without deploying
            </Button>
          }
        >
          <Button type="button">Continue to Service</Button>
        </WizardFooter>
      )}
    </div>
  </StepHeading>
);
