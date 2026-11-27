import { ModelOptionsPicker } from "@/components/model/ModelOptionsPicker";
import { ModelPicker } from "@/components/model/ModelPicker";
import { findModel, type ModelPick } from "@/models/ModelPick";
import type { HarnessProvider, OptionSetting } from "@/models/Pairing";
import { cn } from "@/lib/utils";

// Each half draws focus inside itself, so the ring names the focused half and never crosses the shared frame.
const INNER = "inset-focus h-full rounded-none border-0 bg-transparent shadow-none first:rounded-l-md last:rounded-r-md dark:bg-transparent";

interface ModelChoiceProps {
  providers: HarnessProvider[];
  value: ModelPick;
  options: OptionSetting[];
  onPick: (next: ModelPick) => void;
  onOptions: (next: OptionSetting[]) => void;
  // The model button's accessible name, such as "Claude model"; the options button adds "options".
  label: string;
  allowDefault?: boolean;
  defaultLabel?: string;
  disabled?: boolean;
  mark?: boolean;
  className?: string;
}

// The model button, a hairline, the options' name, then its options button, in one frame; a model with no options shows the model alone.
export const ModelChoice = ({ providers, value, options, onPick, onOptions, label, allowDefault = false, defaultLabel, disabled = false, mark = true, className }: ModelChoiceProps) => {
  const modelOptions = findModel(providers, value)?.options ?? [];
  const frame = "flex h-full items-center rounded-md border border-input shadow-xs dark:bg-input/20";
  return (
    // The model and its options are one right-aligned pair, 8px apart: the model takes the width, Effort about 8rem.
    <div className={cn("inline-flex h-8 min-w-0 items-center gap-2", className)}>
      <div className={cn(frame, "min-w-0 flex-1")}>
        <ModelPicker
          providers={providers}
          value={value}
          onChange={onPick}
          label={label}
          allowDefault={allowDefault}
          {...(defaultLabel && { defaultLabel })}
          disabled={disabled}
          mark={mark}
          className={cn(INNER, "min-w-0 flex-1")}
        />
      </div>
      {modelOptions.length > 0 && (
        <div className={cn(frame, "w-32 shrink-0")}>
          <span aria-hidden className="pl-2.5 text-xs text-muted-foreground">
            Effort
          </span>
          <ModelOptionsPicker options={modelOptions} value={options} onChange={onOptions} label={`${label} options`} disabled={disabled} className={cn(INNER, "min-w-0 flex-1")} />
        </div>
      )}
    </div>
  );
};
