import { XIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { StackEnvFormData } from "@/models/StackEnv";

interface StackEnvironmentRowProps {
  index: number;
  revealed: boolean;
  onRemove: () => void;
}

export const StackEnvironmentRow = ({ index, revealed, onRemove }: StackEnvironmentRowProps) => {
  const { register, control, formState } = useFormContext<StackEnvFormData>();
  const key = useWatch({ control, name: `rows.${index}.key` });
  const message = formState.errors.rows?.[index]?.key?.message;
  const name = key || "new variable";

  return (
    <li className="space-y-1 px-3 py-2">
      <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-center gap-2">
        <Input
          aria-label={`Name of ${name}`}
          aria-invalid={message != null}
          className="font-mono"
          placeholder="NAME"
          spellCheck={false}
          autoCapitalize="off"
          {...register(`rows.${index}.key`)}
        />
        <Input
          aria-label={`Value of ${name}`}
          type={revealed ? "text" : "password"}
          autoComplete="off"
          data-1p-ignore
          className="font-mono"
          placeholder="empty"
          spellCheck={false}
          {...register(`rows.${index}.value`)}
        />
        <Button type="button" variant="ghost" size="icon" aria-label={`Remove ${name}`} onClick={onRemove}>
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
