import { ChevronDownIcon } from "lucide-react";
import { useState } from "react";

import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { ModelPickerPanel } from "@/components/model/ModelPickerPanel";
import { pickerTriggerClass } from "@/components/model/pickerTriggerClass";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { COMPUTER_DEFAULT, findModel, type ModelPick } from "@/models/ModelPick";
import type { HarnessProvider } from "@/models/Pairing";
import { cn } from "@/lib/utils";

interface ModelPickerProps {
  providers: HarnessProvider[];
  value: ModelPick;
  onChange: (next: ModelPick) => void;
  // The accessible name, such as "Claude model".
  label: string;
  // Offers the default first: the provider's own when one provider is in view, else the computer's.
  allowDefault?: boolean;
  // What the empty pick reads as: the computer's default, or the person's defaults inside a project.
  defaultLabel?: string;
  disabled?: boolean;
  // Off where the row beside it already shows the provider's mark.
  mark?: boolean;
  className?: string;
}

// A model button opening a searchable list of every provider's models, as the harness lists them.
export const ModelPicker = ({ providers, value, onChange, label, allowDefault = false, defaultLabel = COMPUTER_DEFAULT, disabled = false, mark = true, className }: ModelPickerProps) => {
  const [open, setOpen] = useState(false);
  const provider = providers.find((p) => p.id === value.provider);
  const text = findModel(providers, value)?.name ?? (value.model || (value.provider ? "Provider default" : defaultLabel));
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger role="combobox" aria-label={label} aria-expanded={open} aria-haspopup="listbox" disabled={disabled} className={cn(pickerTriggerClass, className)}>
        {mark && provider && <HarnessProviderMark driver={provider.driver} className="size-3.5 shrink-0 text-muted-foreground" />}
        <span className={cn("min-w-0 flex-1 truncate text-left", !value.provider && !value.model && "text-muted-foreground")}>{text}</span>
        <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[22rem] max-w-[var(--radix-popover-content-available-width)] p-0">
        <ModelPickerPanel
          providers={providers}
          value={value}
          allowDefault={allowDefault}
          defaultLabel={defaultLabel}
          label={label}
          onPick={(pick) => {
            onChange(pick);
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
};
