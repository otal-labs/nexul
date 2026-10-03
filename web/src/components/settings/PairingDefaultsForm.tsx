import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import {
  PairingDefaultsFormSchema,
  START_IN_OPTIONS,
  StartIn,
  type Computer,
  type PairingDefaults,
  type PairingDefaultsFormData,
} from "@/models/Pairing";

interface PairingDefaultsFormProps {
  defaults: PairingDefaults;
  computers: Computer[];
}

// Split out so defaultValues only seed from a loaded `defaults` — avoids the F2 "= [] trap".
export const PairingDefaultsForm = ({ defaults, computers }: PairingDefaultsFormProps) => {
  const update = useUpdatePairingDefaults();

  const form = useForm<PairingDefaultsFormData>({
    defaultValues: {
      default_computer_id: defaults.default_computer_id ?? "",
      fallback_project_id: defaults.fallback_project_id ?? "",
      provider: defaults.provider ?? "",
      model: defaults.model ?? "",
      model_options: defaults.model_options ?? [],
      start_in: defaults.start_in ?? StartIn.Folder,
    },
    resolver: zodResolver(PairingDefaultsFormSchema),
  });

  const onSubmit = async (data: PairingDefaultsFormData) => {
    try {
      const saved = await update.mutateAsync(data);
      form.reset({
        default_computer_id: saved.default_computer_id ?? "",
        fallback_project_id: saved.fallback_project_id ?? "",
        provider: saved.provider ?? "",
        model: saved.model ?? "",
        model_options: saved.model_options ?? [],
        start_in: saved.start_in ?? StartIn.Folder,
      });
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <FormSelect
        control={form.control}
        name="default_computer_id"
        label="Default computer"
        placeholder="No default"
        options={computers.map((c) => ({ value: c.id, label: c.name }))}
      />
      <HarnessProjectField
        control={form.control}
        name="fallback_project_id"
        label="Fallback T3 project"
        computerId={form.watch("default_computer_id")}
      />
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        optionsName="model_options"
        computerId={form.watch("default_computer_id")}
        description="Default model for chats and runs with no project link"
        onPick={(provider, model, options) => {
          form.setValue("provider", provider, { shouldDirty: true });
          form.setValue("model", model, { shouldDirty: true });
          form.setValue("model_options", options, { shouldDirty: true });
        }}
      />
      <FormSelect control={form.control} name="start_in" label="New threads start in" options={START_IN_OPTIONS} />
      <Button type="submit" loading={form.formState.isSubmitting}>
        Save defaults
      </Button>
    </form>
  );
};
