import { ModelOptionsPicker } from "@/components/model/ModelOptionsPicker";
import { ModelPicker } from "@/components/model/ModelPicker";
import { findModel, type ModelPick } from "@/models/ModelPick";
import type { HarnessProvider, OptionSetting } from "@/models/Pairing";
import { cn } from "@/lib/utils";

const INNER = "h-full rounded-none border-0 bg-transparent shadow-none first:rounded-l-md last:rounded-r-md dark:bg-transparent";

interface ModelChoiceProps {
  providers: HarnessProvider[];
  value: ModelPick;
  options: OptionSetting[];
  onPick: (next: ModelPick) => void;
  onOptions: (next: OptionSetting[]) => void;
  // The model button's accessible name, such as "Claude model"; the options button adds "options".
  label: string;
  allowDefault?: boolean;
  disabled?: boolean;
  className?: string;
}

// The model button, a hairline, then its options button, in one frame; a model with no options shows the model alone.
export const ModelChoice = ({ providers, value, options, onPick, onOptions, label, allowDefault = false, disabled = false, className }: ModelChoiceProps) => {
  const modelOptions = findModel(providers, value)?.options ?? [];
  return (
    <div className={cn("inline-flex h-8 min-w-0 items-center rounded-md border border-input shadow-xs dark:bg-input/20", className)}>
      <ModelPicker
        providers={providers}
        value={value}
        onChange={onPick}
        label={label}
        allowDefault={allowDefault}
        disabled={disabled}
        className={cn(INNER, "min-w-0 flex-1")}
      />
      {modelOptions.length > 0 && <span aria-hidden className="h-4 w-px shrink-0 bg-border" />}
      <ModelOptionsPicker options={modelOptions} value={options} onChange={onOptions} label={`${label} options`} disabled={disabled} className={cn(INNER, "shrink-0")} />
    </div>
  );
};
