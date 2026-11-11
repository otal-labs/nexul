import { zodResolver } from "@hookform/resolvers/zod";
import { EyeIcon, EyeOffIcon, PlusIcon } from "lucide-react";
import { FormProvider, useFieldArray, useForm } from "react-hook-form";
import { toast } from "sonner";

import { holderNamedBy, isReferenceRefusal } from "@nexul/client-core/secretReference";

import { errorMessage } from "@/api/client";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { StackDeployNowButton } from "@/components/stack/StackDeployNowButton";
import { StackEnvironmentRow } from "@/components/stack/StackEnvironmentRow";
import { Button } from "@/components/ui/button";
import { useUpdateStackEnv } from "@/hooks/StackHooks";
import type { Stack } from "@/models/Stack";
import { envToRows, rowsToEnv, StackEnvFormSchema, type StackEnvFormData } from "@/models/StackEnv";

const FORM_ID = "stack-environment-form";

interface StackEnvironmentFormProps {
  stack: Stack;
  image: string | undefined;
  revealed: boolean;
  onReveal: (revealed: boolean) => void;
  saved: boolean;
  onSaved: () => void;
}

export const StackEnvironmentForm = ({ stack, image, revealed, onReveal, saved, onSaved }: StackEnvironmentFormProps) => {
  const updateEnv = useUpdateStackEnv({ quiet: true, inline: true });
  const form = useForm<StackEnvFormData>({
    defaultValues: { rows: envToRows(stack.env) },
    resolver: zodResolver(StackEnvFormSchema),
  });
  const { fields, append, remove } = useFieldArray({ control: form.control, name: "rows" });

  const onSubmit = async (data: StackEnvFormData) => {
    try {
      await updateEnv.mutateAsync({ ...stack, env: rowsToEnv(data.rows) });
      onSaved();
    } catch (err) {
      // The hook toasts any other error; a refused reference shows under its row, and the edits stay in the form to retry.
      const message = errorMessage(err);
      const holder = holderNamedBy(message, data.rows.map((row) => row.key));
      const index = data.rows.findIndex((row) => row.key === holder);
      if (index >= 0) form.setError(`rows.${index}.value`, { message });
      if (index < 0 && isReferenceRefusal(message)) toast.error(message);
    }
  };

  return (
    <SettingsCard
      id="environment"
      title="Environment"
      description="Saved changes apply on the next deploy. Saving restarts nothing."
      aside={
        <Button type="button" variant="ghost" size="sm" aria-pressed={revealed} onClick={() => onReveal(!revealed)}>
          {revealed && <EyeOffIcon aria-hidden />}
          {!revealed && <EyeIcon aria-hidden />}
          {revealed ? "Hide values" : "Reveal values"}
        </Button>
      }
      footer={
        <>
          <SettingsSaveBar
            form={FORM_ID}
            dirty={form.formState.isDirty}
            saving={form.formState.isSubmitting}
            saved={saved}
            onDiscard={() => form.reset()}
          />
          <StackDeployNowButton stack={stack} image={image} disabled={form.formState.isDirty} />
        </>
      }
    >
      <FormProvider {...form}>
        <form id={FORM_ID} onSubmit={form.handleSubmit(onSubmit)} className="space-y-3">
          {fields.length === 0 && <EmptyRow flush>No variables are set.</EmptyRow>}
          {fields.length > 0 && (
            <ul className="divide-y divide-border rounded-md border border-border">
              {fields.map((field, index) => (
                <StackEnvironmentRow key={field.id} index={index} revealed={revealed} onRemove={() => remove(index)} />
              ))}
            </ul>
          )}
          <Button type="button" variant="ghost" size="sm" onClick={() => append({ key: "", value: "" }, { shouldFocus: true })}>
            <PlusIcon aria-hidden />
            Add variable
          </Button>
        </form>
      </FormProvider>
    </SettingsCard>
  );
};
