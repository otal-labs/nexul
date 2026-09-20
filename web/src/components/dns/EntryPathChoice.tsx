import { useState } from "react";

import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { entryPathOptions, type EntryPath, type EntryPathOption } from "@/models/DNS";

interface EntryPathRowProps {
  option: EntryPathOption;
}

// A row with a hairline divider, per the design language; cards are for draggable units only.
const EntryPathRow = ({ option }: EntryPathRowProps) => (
  <label
    htmlFor={`entry-path-${option.value}`}
    className="-mx-2 flex cursor-pointer items-start gap-3 rounded-md px-2 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40"
  >
    <RadioGroupItem id={`entry-path-${option.value}`} value={option.value} className="mt-0.5" />
    <span className="min-w-0">
      <span className="block text-sm font-medium">{option.label}</span>
      <span className="mt-0.5 block text-sm text-muted-foreground">{option.description}</span>
    </span>
  </label>
);

interface EntryPathChoiceProps {
  onContinue: (path: EntryPath) => void;
}

export const EntryPathChoice = ({ onContinue }: EntryPathChoiceProps) => {
  const [path, setPath] = useState<EntryPath | null>(null);

  return (
    <div className="space-y-5">
      <RadioGroup
        aria-label="Entry path"
        value={path ?? ""}
        onValueChange={(value) => setPath(value as EntryPath)}
        className="gap-0 divide-y divide-border border-y border-border"
      >
        {entryPathOptions.map((option) => (
          <EntryPathRow key={option.value} option={option} />
        ))}
      </RadioGroup>
      <Button className="w-full sm:w-auto" disabled={path === null} onClick={() => path && onContinue(path)}>
        Continue
      </Button>
    </div>
  );
};
