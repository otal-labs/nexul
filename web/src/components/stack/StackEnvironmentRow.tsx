import { XIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

import { SecretReferenceFieldTag } from "@/components/SecretReferenceFieldTag";
import { TaggedInput } from "@/components/TaggedInput";
import { envGridClass } from "@/components/stack/StackEnvironmentHead";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { StackEnvFormData } from "@/models/StackEnv";

interface StackEnvironmentRowProps {
  index: number;
  revealed: boolean;
  onRemove: () => void;
}

// Name and value read as mono text in a row until hovered or focused, when the value's well appears to type into.
const keyFieldClass = "border-transparent bg-transparent px-2 font-mono shadow-none dark:bg-transparent";
const valueFieldClass =
  "border-transparent bg-transparent font-mono shadow-none transition-colors duration-150 ease-standard dark:bg-transparent hover:bg-surface-2 focus-visible:bg-surface-2 dark:hover:bg-surface-2 dark:focus-visible:bg-surface-2";

export const StackEnvironmentRow = ({ index, revealed, onRemove }: StackEnvironmentRowProps) => {
  const { register, control, formState } = useFormContext<StackEnvFormData>();
  const key = useWatch({ control, name: `rows.${index}.key` });
  const value = useWatch({ control, name: `rows.${index}.value` });
  // A masked value shows one fixed run of dots until focused, so the column never hints at a secret's length.
  const masked = !revealed && !!value;
  const rowErrors = formState.errors.rows?.[index];
  const message = rowErrors?.key?.message ?? rowErrors?.value?.message;
  const name = key || "new variable";

  return (
    <li className="group space-y-1 py-1.5">
      <div className={envGridClass}>
        <Input
          aria-label={`Name of ${name}`}
          aria-invalid={rowErrors?.key != null}
          className={keyFieldClass}
          placeholder="NAME"
          spellCheck={false}
          autoCapitalize="off"
          {...register(`rows.${index}.key`)}
        />
        <TaggedInput tag={<SecretReferenceFieldTag control={control} name={`rows.${index}.value`} />}>
          <Input
            aria-label={`Value of ${name}`}
            aria-invalid={rowErrors?.value != null}
            type={revealed ? "text" : "password"}
            autoComplete="off"
            data-1p-ignore
            className={cn("peer", valueFieldClass, masked && "text-transparent focus:text-foreground")}
            placeholder="empty"
            spellCheck={false}
            {...register(`rows.${index}.value`)}
          />
          {masked && (
            <span aria-hidden className="pointer-events-none absolute inset-y-0 left-3 flex items-center font-mono text-sm text-muted-foreground peer-focus:hidden">
              ••••••••••••
            </span>
          )}
        </TaggedInput>
        <Button type="button" variant="ghost" size="icon" aria-label={`Remove ${name}`} onClick={onRemove} className="size-8 text-muted-foreground opacity-0 transition-[color,background-color,opacity] duration-150 ease-standard group-focus-within:opacity-100 group-hover:opacity-100 hover:text-foreground pointer-coarse:opacity-100">
          <XIcon aria-hidden />
        </Button>
      </div>
      {message && (
        <p role="alert" className="text-sm text-destructive">
          {message}
        </p>
      )}
    </li>
  );
};
