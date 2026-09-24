import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { SetupModelChoice } from "@/models/Pairing";

// Radix Select reserves the empty value, so the provider's own default travels under this one.
const PROVIDER_DEFAULT = "provider-default";

interface SetupModelPickProps {
  choice: SetupModelChoice;
  value: string;
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
}

const SetupModelPick = ({ choice, value, disabled, onPick }: SetupModelPickProps) => {
  const id = `setup-model-${choice.provider}`;
  return (
    <div className="flex flex-col gap-1.5 @sm:flex-row @sm:items-center @sm:justify-between @sm:gap-3">
      <label htmlFor={id} className="min-w-0 text-sm break-words">
        {choice.name}
      </label>
      <Select
        value={value || PROVIDER_DEFAULT}
        onValueChange={(v) => onPick(choice.provider, v === PROVIDER_DEFAULT ? "" : v)}
        disabled={disabled}
      >
        <SelectTrigger id={id} className="h-8 @sm:w-60">
          <span className="min-w-0 truncate">
            <SelectValue />
          </span>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={PROVIDER_DEFAULT}>Provider default</SelectItem>
          {choice.models.map((m) => (
            <SelectItem key={m.slug} value={m.slug}>
              {m.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};

interface SetupModelPicksProps {
  choices: SetupModelChoice[];
  models: Record<string, string>;
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
}

// The model each provider's setup turn runs on, as T3 Code lists them on this computer.
export const SetupModelPicks = ({ choices, models, disabled, onPick }: SetupModelPicksProps) => (
  <fieldset className="@container space-y-2.5">
    <legend className="mb-2.5 text-xs font-semibold">Models</legend>
    {choices.map((c) => (
      <SetupModelPick key={c.provider} choice={c} value={models[c.provider] ?? ""} disabled={disabled} onPick={onPick} />
    ))}
  </fieldset>
);
