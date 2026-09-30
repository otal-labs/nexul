import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { ModelChoice } from "@/components/model/ModelChoice";
import { Switch } from "@/components/ui/switch";

import type { OptionSetting, SetupModelChoice } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface SetupModelPickProps {
  choice: SetupModelChoice;
  value: string;
  options: OptionSetting[];
  included: boolean;
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
  onOptions: (provider: string, options: OptionSetting[]) => void;
  onInclude: (provider: string, on: boolean) => void;
}

const SetupModelPick = ({ choice, value, options, included, disabled, onPick, onOptions, onInclude }: SetupModelPickProps) => (
  <div className="flex min-w-0 items-center gap-2.5">
    <Switch aria-label={choice.name} checked={included} onCheckedChange={(on) => onInclude(choice.provider, on)} disabled={disabled} />
    <HarnessProviderMark driver={choice.instance.driver} className={cn("size-4 shrink-0", !included && "opacity-50")} />
    <ModelChoice
      providers={[choice.instance]}
      value={{ provider: choice.instance.id, model: value }}
      options={options}
      onPick={(next) => onPick(choice.provider, next.model)}
      onOptions={(next) => onOptions(choice.provider, next)}
      label={`${choice.name} model`}
      allowDefault
      disabled={disabled || !included}
      className="flex-1"
    />
  </div>
);

interface SetupModelPicksProps {
  choices: SetupModelChoice[];
  models: Record<string, string>;
  options: Record<string, OptionSetting[]>;
  included: string[];
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
  onOptions: (provider: string, options: OptionSetting[]) => void;
  onInclude: (provider: string, on: boolean) => void;
}

// Each provider T3 Code lists on this computer: a switch for whether setup covers it, and the model and options its turn runs on.
export const SetupModelPicks = ({ choices, models, options, included, disabled, onPick, onOptions, onInclude }: SetupModelPicksProps) => (
  <fieldset className="@container space-y-3">
    <legend className="mb-3 text-xs font-semibold">Models</legend>
    {choices.map((c) => (
      <SetupModelPick
        key={c.provider}
        choice={c}
        value={models[c.provider] ?? ""}
        options={options[c.provider] ?? []}
        included={included.includes(c.provider)}
        disabled={disabled}
        onPick={onPick}
        onOptions={onOptions}
        onInclude={onInclude}
      />
    ))}
  </fieldset>
);
