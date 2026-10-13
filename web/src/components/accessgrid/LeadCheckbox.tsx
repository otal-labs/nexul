import { Check, Minus } from "lucide-react";
import { Checkbox as CheckboxPrimitive } from "radix-ui";

interface LeadCheckboxProps {
  checked: boolean | "indeterminate";
  onCheckedChange: (checked: boolean) => void;
  label: string;
}

// The lead row's box: ticked when every row has it, a dash when some do; a click ticks all, or clears all from ticked.
export const LeadCheckbox = ({ checked, onCheckedChange, label }: LeadCheckboxProps) => (
  <CheckboxPrimitive.Root
    checked={checked}
    aria-label={label}
    onCheckedChange={() => onCheckedChange(checked !== true)}
    className="peer size-4 shrink-0 rounded-xs border border-input shadow-xs outline-none transition-[background-color,border-color] duration-150 ease-standard data-[state=checked]:border-brand data-[state=checked]:bg-brand data-[state=checked]:text-brand-foreground data-[state=indeterminate]:border-foreground/40 data-[state=indeterminate]:text-foreground"
  >
    <CheckboxPrimitive.Indicator className="flex items-center justify-center text-current">
      {checked === "indeterminate" && <Minus className="size-3.5" />}
      {checked === true && <Check className="size-3.5" />}
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
);
