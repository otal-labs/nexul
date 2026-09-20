import type { ComponentProps } from "react";
import {
  Controller,
  useFormState,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

import { Input } from "@/components/ui/input";

interface FormInputProps<T extends FieldValues>
  extends Omit<ComponentProps<"input">, "name"> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  id?: string;
  // Keeps the accessible name but hides it visually, for a field whose purpose is obvious from context.
  hideLabel?: boolean;
}

export const FormInput = <T extends FieldValues>({
  control,
  name,
  label,
  id,
  hideLabel,
  ...props
}: FormInputProps<T>) => {
  const { errors } = useFormState({ control });
  const message = (errors[name]?.message as string | undefined) ?? undefined;
  const fieldId = id ?? name;

  return (
    <div className="space-y-2">
      <label htmlFor={fieldId} className={hideLabel ? "sr-only" : "text-sm font-medium"}>
        {label}
      </label>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Input id={fieldId} aria-invalid={message != null} {...props} {...field} />
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
