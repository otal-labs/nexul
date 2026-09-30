import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

import type { SetupModelChoice } from "@/models/Pairing";
import { cn } from "@/lib/utils";

// Radix Select reserves the empty value, so the provider's own default travels under this one.
const PROVIDER_DEFAULT = "provider-default";

interface SetupModelPickProps {
  choice: SetupModelChoice;
  value: string;
  included: boolean;
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
  onInclude: (provider: string, on: boolean) => void;
}

const SetupModelPick = ({ choice, value, included, disabled, onPick, onInclude }: SetupModelPickProps) => {
  const id = `setup-include-${choice.provider}`;
  return (
    <div className="flex flex-col gap-2 @xs:flex-row @xs:items-center @xs:justify-between @xs:gap-3">
      <div className="flex min-w-0 items-center gap-2.5">
        <Switch id={id} checked={included} onCheckedChange={(on) => onInclude(choice.provider, on)} disabled={disabled} />
        <label htmlFor={id} className={cn("min-w-0 text-sm break-words", !included && "text-muted-foreground")}>
          {choice.name}
        </label>
      </div>
      <Select
        value={value || PROVIDER_DEFAULT}
        onValueChange={(v) => onPick(choice.provider, v === PROVIDER_DEFAULT ? "" : v)}
        disabled={disabled || !included}
      >
        <SelectTrigger aria-label={`${choice.name} model`} className="h-8 @xs:w-52">
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
  included: string[];
  disabled: boolean;
  onPick: (provider: string, model: string) => void;
  onInclude: (provider: string, on: boolean) => void;
}

// Each provider T3 Code lists on this computer: a switch for whether setup covers it, and the model its turn runs on.
export const SetupModelPicks = ({ choices, models, included, disabled, onPick, onInclude }: SetupModelPicksProps) => (
  <fieldset className="@container space-y-3">
    <legend className="mb-3 text-xs font-semibold">Models</legend>
    {choices.map((c) => (
      <SetupModelPick
        key={c.provider}
        choice={c}
        value={models[c.provider] ?? ""}
        included={included.includes(c.provider)}
        disabled={disabled}
        onPick={onPick}
        onInclude={onInclude}
      />
    ))}
  </fieldset>
);
