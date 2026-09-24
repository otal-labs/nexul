import {
  Controller,
  get,
  useFormState,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface FormSelectOption {
  value: string;
  label: string;
}

interface FormSelectProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  options: FormSelectOption[];
  placeholder?: string;
  // Keeps the accessible name (aria-labelledby) but hides it visually when purpose is obvious.
  hideLabel?: boolean;
  /** Side-effect on selection (e.g. deriving a sibling field) — the form value is already handled. */
  onChangeValue?: (value: string) => void;
}

// Radix bans an empty-string value; this sentinel keeps "" (uncategorized) selectable.
const NONE = "__none__";

export const FormSelect = <T extends FieldValues>({
  control,
  name,
  label,
  options,
  placeholder,
  hideLabel,
  onChangeValue,
}: FormSelectProps<T>) => {
  const { errors } = useFormState({ control });
  const message = (get(errors, name)?.message as string | undefined) ?? undefined;
  const labelId = `${name}-label`;

  return (
    <div className="space-y-2">
      <label id={labelId} className={hideLabel ? "sr-only" : "text-sm font-medium"}>
        {label}
      </label>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Select
            value={field.value as string}
            onValueChange={(value) => {
              const next = value === NONE ? "" : value;
              field.onChange(next);
              onChangeValue?.(next);
            }}
          >
            <SelectTrigger id={name} aria-labelledby={labelId} aria-invalid={message != null} onBlur={field.onBlur}>
              <SelectValue placeholder={placeholder ?? "Select…"} />
            </SelectTrigger>
            <SelectContent>
              {placeholder && <SelectItem value={NONE}>{placeholder}</SelectItem>}
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
      />
      {message && (
        <p
          role="alert"
          className="animate-in fade-in-0 slide-in-from-top-0.5 text-sm text-destructive duration-150 ease-out"
        >
          {message}
        </p>
      )}
    </div>
  );
};
