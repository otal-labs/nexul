import { ChevronDownIcon } from "lucide-react";

import { pickerTriggerClass } from "@/components/model/pickerTriggerClass";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { optionsLabel, optionValue, setOption } from "@/models/ModelPick";
import type { ModelOption, OptionChoice, OptionSetting } from "@/models/Pairing";
import { cn } from "@/lib/utils";

// A switch reads as an On and Off pair of choices, so every section is one radio group.
const sectionChoices = (option: ModelOption): OptionChoice[] => {
  if (option.type === "boolean") {
    return [
      { id: "on", label: "On", is_default: !!option.default_on },
      { id: "off", label: "Off", is_default: !option.default_on },
    ];
  }
  return option.choices ?? [];
};

const sectionValue = (option: ModelOption, value: string | boolean | undefined) => {
  if (option.type === "boolean") return value === true ? "on" : "off";
  return typeof value === "string" ? value : "";
};

interface ModelOptionSectionProps {
  option: ModelOption;
  separated: boolean;
  value: string | boolean | undefined;
  onSelect: (value: string | boolean) => void;
}

const ModelOptionSection = ({ option, separated, value, onSelect }: ModelOptionSectionProps) => (
  <DropdownMenuGroup>
    {separated && <DropdownMenuSeparator />}
    <DropdownMenuLabel>{option.label}</DropdownMenuLabel>
    <DropdownMenuRadioGroup
      value={sectionValue(option, value)}
      onValueChange={(id) => onSelect(option.type === "boolean" ? id === "on" : id)}
    >
      {sectionChoices(option).map((choice) => (
        <DropdownMenuRadioItem key={choice.id} value={choice.id} className="items-start">
          <span className="flex min-w-0 flex-1 flex-col">
            <span>{choice.label}</span>
            {choice.description && <span className="text-xs text-muted-foreground">{choice.description}</span>}
          </span>
          {choice.is_default && <span className="text-xs text-muted-foreground">Default</span>}
        </DropdownMenuRadioItem>
      ))}
    </DropdownMenuRadioGroup>
  </DropdownMenuGroup>
);

interface ModelOptionsPickerProps {
  // The picked model's options as the harness lists them; none renders nothing.
  options: ModelOption[];
  value: OptionSetting[];
  onChange: (next: OptionSetting[]) => void;
  label: string;
  disabled?: boolean;
  className?: string;
}

// The button beside a model, such as "High · 1M", opening one section per option the model supports, defaults marked.
export const ModelOptionsPicker = ({ options, value, onChange, label, disabled = false, className }: ModelOptionsPickerProps) => {
  if (options.length === 0) return null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger aria-label={label} disabled={disabled} className={cn(pickerTriggerClass, className)}>
        <span className="min-w-0 truncate">{optionsLabel(options, value)}</span>
        <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        {options.map((option, index) => (
          <ModelOptionSection
            key={option.id}
            option={option}
            separated={index > 0}
            value={optionValue(option, value)}
            onSelect={(next) => onChange(setOption(value, option.id, next))}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
