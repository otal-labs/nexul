import { ChevronDownIcon } from "lucide-react";

import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { EmptyRow } from "@/components/EmptyRow";
import { menuItemClass } from "@/components/MenuItem";
import type { FieldValues } from "@/hooks/AutoPlayValueHooks";
import { cn } from "@/lib/utils";

interface ValuesPickerProps {
  label: string;
  values: string[];
  field: FieldValues;
  onChange: (values: string[]) => void;
  disabled?: boolean | undefined;
}

// Several values for is / is not: reads like a select, holds a check per value; a picked value no option names stays listed.
export const ValuesPicker = ({ label, values, field, onChange, disabled }: ValuesPickerProps) => {
  const unknown = values.filter((value) => !field.options.some((option) => option.value === value));
  const rows = [...unknown.map((value) => ({ value, label: field.label(value) })), ...field.options];
  const text = values.map(field.label).join(", ");
  const toggle = (value: string) =>
    onChange(values.includes(value) ? values.filter((v) => v !== value) : [...values, value]);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={label}
          disabled={disabled}
          title={text}
          className="flex h-8 min-w-0 flex-1 items-center justify-between gap-1.5 rounded-md border border-input px-2.5 text-left text-sm shadow-xs transition-[border-color] duration-150 ease-standard hover:border-foreground/25 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/20"
        >
          <span className={cn("truncate", values.length === 0 && "text-muted-foreground")}>{text || "Pick values"}</span>
          <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="max-h-72 w-64 overflow-y-auto p-1">
        {rows.map((row) => (
          <label key={row.value} className={cn(menuItemClass, "cursor-pointer")} title={row.label}>
            <Checkbox checked={values.includes(row.value)} onCheckedChange={() => toggle(row.value)} />
            <span className="truncate">{row.label}</span>
          </label>
        ))}
        {rows.length === 0 && (
          <EmptyRow flush className="px-2 py-1.5">
            Nothing to pick yet
          </EmptyRow>
        )}
      </PopoverContent>
    </Popover>
  );
};
