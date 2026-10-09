import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";

import { FormSelect } from "@/components/ticket/FormSelect";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import { useFlash } from "@/hooks/useFlash";
import {
  PairingDefaultsFormSchema,
  START_IN_OPTIONS,
  StartIn,
  type Computer,
  type PairingDefaults,
  type PairingDefaultsFormData,
} from "@/models/Pairing";

const FORM_ID = "pairing-defaults-form";

export const DEFAULTS_DESCRIPTION = "What @Agent uses outside a project, and in any project you haven't linked on the Projects tab.";

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
  const defaultComputerId = useWatch({ control: form.control, name: "default_computer_id" });

  const [saved, flash] = useFlash();

  const onSubmit = async (data: PairingDefaultsFormData) => {
    try {
      const stored = await update.mutateAsync(data);
      form.reset({
        default_computer_id: stored.default_computer_id ?? "",
        fallback_project_id: stored.fallback_project_id ?? "",
        provider: stored.provider ?? "",
        model: stored.model ?? "",
        model_options: stored.model_options ?? [],
        start_in: stored.start_in ?? StartIn.Folder,
      });
      flash();
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <SettingsCard
      id="pairing-defaults"
      title="Defaults"
      description={DEFAULTS_DESCRIPTION}
      footer={
        <SettingsSaveBar
          form={FORM_ID}
          dirty={form.formState.isDirty}
          saving={form.formState.isSubmitting}
          saved={saved}
          onDiscard={() => form.reset()}
          saveLabel="Save defaults"
        />
      }
    >
    <form id={FORM_ID} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
        computerId={defaultComputerId}
      />
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        optionsName="model_options"
        computerId={defaultComputerId}
        description="Default model for chats and runs with no project link"
        onPick={(provider, model, options) => {
          form.setValue("provider", provider, { shouldDirty: true });
          form.setValue("model", model, { shouldDirty: true });
          form.setValue("model_options", options, { shouldDirty: true });
        }}
      />
      <FormSelect control={form.control} name="start_in" label="New threads start in" options={START_IN_OPTIONS} />
    </form>
    </SettingsCard>
  );
};
