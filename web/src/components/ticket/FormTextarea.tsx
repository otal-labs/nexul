import type { ComponentProps } from "react";
import {
  Controller,
  get,
  useFormState,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

import { cn } from "@/lib/utils";

interface FormTextareaProps<T extends FieldValues>
  extends Omit<ComponentProps<"textarea">, "name"> {
  control: Control<T>;
  name: Path<T>;
  label: string;
}

export const FormTextarea = <T extends FieldValues>({
  control,
  name,
  label,
  className,
  ...props
}: FormTextareaProps<T>) => {
  const { errors } = useFormState({ control });
  const message = (get(errors, name)?.message as string | undefined) ?? undefined;

  return (
    <div className="space-y-2">
      <label htmlFor={name} className="text-sm font-medium">
        {label}
      </label>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <textarea
            id={name}
            className={cn("w-full rounded-md border px-3 py-2 text-sm", className)}
            aria-invalid={message != null}
            {...props}
            {...field}
          />
        )}
      />
      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
    </div>
  );
};
