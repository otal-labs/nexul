import type { ComponentProps, ReactNode } from "react";
import {
  Controller,
  get,
  useFormState,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";

import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface FormInputProps<T extends FieldValues>
  extends Omit<ComponentProps<"input">, "name"> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  id?: string;
  // Keeps the accessible name but hides it visually, for a field whose purpose is obvious from context.
  hideLabel?: boolean;
  // Rewrites the value as the user types, so the stored value is already normalized.
  transform?: (value: string) => string;
  // Called with the new text after the field takes it, so a sibling field can follow along.
  onValueChange?: (value: string) => void;
  // Fixed text joined to the front of the input, such as the address a slug follows.
  leading?: ReactNode;
  // Marks the field wrong for a reason outside the form's own rules, such as a check the server failed.
  invalid?: boolean;
}

export const FormInput = <T extends FieldValues>({
  control,
  name,
  label,
  id,
  hideLabel,
  transform,
  onValueChange,
  leading,
  invalid = false,
  className,
  ...props
}: FormInputProps<T>) => {
  const { errors } = useFormState({ control });
  const message = (get(errors, name)?.message as string | undefined) ?? undefined;
  const fieldId = id ?? name;

  const input = (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <Input
          id={fieldId}
          aria-invalid={message != null || invalid}
          {...props}
          {...field}
          // Joined to a prefix, the group draws the border and the focus ring for both.
          className={cn(leading && "quiet-focus h-full rounded-none border-0 shadow-none dark:bg-transparent", className)}
          onChange={(e) => {
            field.onChange(transform?.(e.target.value) ?? e);
            onValueChange?.(e.target.value);
          }}
        />
      )}
    />
  );

  return (
    <div className="space-y-2">
      <label htmlFor={fieldId} className={hideLabel ? "sr-only" : "text-sm font-medium"}>
        {label}
      </label>
      {leading && (
        <div className="flex h-9 overflow-hidden rounded-md border border-input shadow-xs transition-[border-color] duration-150 ease-standard hover:border-foreground/25 has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-1 has-[:focus-visible]:outline-(--focus) has-[[aria-invalid=true]]:border-destructive dark:bg-input/20">
          <span className="flex shrink-0 items-center border-r border-input bg-surface-2 px-3 font-mono text-sm text-muted-foreground">
            {leading}
          </span>
          {input}
        </div>
      )}
      {!leading && input}
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
