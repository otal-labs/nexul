import { useState } from "react";

import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { TestsLocationChoice } from "@/components/wizard/TestsLocationChoice";
import { WizardFooter } from "@/components/wizard/WizardFooter";

import { PickedSummary, SectionLabel, StepHeading, type RepoStepProps } from "./parts";
import { SearchMock } from "./SearchMock";

type Way = "deploy" | "attach" | "none";

const ways: { value: Way; label: string; description: string }[] = [
  { value: "deploy", label: "Deploy a repository", description: "Nexul reads its Dockerfile or compose file and sets up a service." },
  { value: "attach", label: "Attach a repository, deploy later", description: "Its pull requests link to tickets now. Deploy it once it's ready." },
  { value: "none", label: "No repository yet", description: "Start with docs and tickets. Attach one any time from the project's Settings." },
];

const WayRow = ({ value, label, description }: (typeof ways)[number]) => (
  <label
    htmlFor={`way-${value}`}
    className="-mx-2 flex cursor-pointer items-start gap-3 rounded-md px-2 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40"
  >
    <RadioGroupItem id={`way-${value}`} value={value} className="mt-0.5" />
    <span className="min-w-0">
      <span className="block text-sm font-medium">{label}</span>
      <span className="mt-0.5 block text-sm text-muted-foreground">{description}</span>
    </span>
  </label>
);

const primaryLabel = (way: Way, picked: boolean): string => {
  if (way === "none") return "Continue to Done";
  if (way === "attach") return "Attach and continue to Done";
  return picked ? "Continue to Service" : "Pick a repository";
};

// Variant 3: how the repository comes in is the step's first question, asked as three equal rows.
export const RepoStepChoice = ({ picked, onPick, onUnpick, onExit }: RepoStepProps) => {
  const [way, setWay] = useState<Way>(() => (new URLSearchParams(location.search).get("way") as Way | null) ?? "deploy");
  const needsRepo = way !== "none";
  const ready = way === "none" || picked;
  const go = () => {
    if (way === "none") onExit("none");
    if (way === "attach" && picked) onExit("attached");
  };

  return (
    <StepHeading description="Does this project have code yet? Say how its repository comes in; you can change it later.">
      <div className="mt-4 space-y-5">
        <RadioGroup
          aria-label="How the repository comes in"
          value={way}
          onValueChange={(value) => setWay(value as Way)}
          className="gap-0 divide-y divide-border border-y border-border"
        >
          {ways.map((w) => (
            <WayRow key={w.value} {...w} />
          ))}
        </RadioGroup>
        <TestsLocationChoice />
        {needsRepo && <SectionLabel>Repository</SectionLabel>}
        {needsRepo && !picked && <SearchMock onPick={onPick} />}
        {needsRepo && picked && <PickedSummary onChange={onUnpick} />}
        <WizardFooter onBack={() => undefined}>
          <Button type="button" disabled={!ready} onClick={go}>
            {primaryLabel(way, picked)}
          </Button>
        </WizardFooter>
      </div>
    </StepHeading>
  );
};
