import { useState } from "react";

import { SettingsHint } from "@/components/settings/SettingsHint";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { InWizardBar } from "@/components/wizard/InWizardBar";
import { wizardOptionClass } from "@/components/wizard/WizardCardList";
import { useWizardBar } from "@/hooks/useWizardBar";
import { cn } from "@/lib/utils";
import { entryPathOptions, type EntryPath, type EntryPathOption } from "@/models/DNS";

interface EntryPathRowProps {
  option: EntryPathOption;
}

// A hairline row in a settings flow; inside a wizard, the wizards' choice card.
const EntryPathRow = ({ option }: EntryPathRowProps) => {
  const inWizard = !!useWizardBar();
  return (
    <label
      htmlFor={`entry-path-${option.value}`}
      className={cn(
        !inWizard && "flex cursor-pointer items-start gap-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent",
        inWizard && wizardOptionClass,
      )}
    >
      <RadioGroupItem id={`entry-path-${option.value}`} value={option.value} className="mt-0.5" />
      <span className="min-w-0">
        <span className="block text-sm font-medium">{option.label}</span>
        <SettingsHint className="mt-0.5">{option.description}</SettingsHint>
      </span>
    </label>
  );
};

interface EntryPathChoiceProps {
  options?: EntryPathOption[];
  onContinue: (path: EntryPath) => void;
}

export const EntryPathChoice = ({ options = entryPathOptions, onContinue }: EntryPathChoiceProps) => {
  const [path, setPath] = useState<EntryPath | null>(null);
  const inWizard = !!useWizardBar();

  return (
      <div className="space-y-5">
        <RadioGroup
          aria-label="Entry path"
          value={path ?? ""}
          onValueChange={(value) => setPath(value as EntryPath)}
          className={cn(inWizard && "gap-2", !inWizard && "gap-0 divide-y divide-border border-y border-border")}
        >
          {options.map((option) => (
            <EntryPathRow key={option.value} option={option} />
          ))}
        </RadioGroup>
        <InWizardBar>
          <Button size={inWizard ? "default" : "lg"} className={inWizard ? "w-full sm:w-auto" : "w-full"} disabled={path === null} onClick={() => path && onContinue(path)}>
            Continue
          </Button>
        </InWizardBar>
      </div>
  );
};
