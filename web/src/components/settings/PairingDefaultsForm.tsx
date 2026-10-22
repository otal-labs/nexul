import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormSelect } from "@/components/ticket/FormSelect";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import { useFlash } from "@/hooks/useFlash";
import {
  PairingDefaultsFormSchema,
  PLACEMENT_OPTIONS,
  START_IN_OPTIONS,
  StartIn,
  type PairingDefaults,
  type PairingDefaultsFormData,
} from "@/models/Pairing";

const FORM_ID = "pairing-defaults-form";

export const DEFAULTS_DESCRIPTION = "Used unless a project sets its own.";

const formValues = (d: PairingDefaults): PairingDefaultsFormData => ({
  provider: d.provider ?? "",
  model: d.model ?? "",
  model_options: d.model_options ?? [],
  start_in: d.start_in ?? StartIn.Folder,
  placement: d.placement,
  clone_automatically: d.clone_automatically,
});

interface PairingDefaultsFormProps {
  defaults: PairingDefaults;
  // The computer whose models the picker lists: the first in the person's order.
  computerId: string;
}

// Split out so defaultValues only seed from a loaded `defaults`, avoiding the F2 "= [] trap".
export const PairingDefaultsForm = ({ defaults, computerId }: PairingDefaultsFormProps) => {
  const update = useUpdatePairingDefaults();
  const form = useForm<PairingDefaultsFormData>({ defaultValues: formValues(defaults), resolver: zodResolver(PairingDefaultsFormSchema) });
  const [saved, flash] = useFlash();

  const onSubmit = async (data: PairingDefaultsFormData) => {
    try {
      form.reset(formValues(await update.mutateAsync(data)));
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
      <form id={FORM_ID} onSubmit={form.handleSubmit(onSubmit)}>
        <SettingsRows>
          <SettingsRow label="Send runs to">
            <FormSelect control={form.control} name="placement" label="Send runs to" hideLabel options={PLACEMENT_OPTIONS} />
          </SettingsRow>
          <div className="py-4">
            <HarnessProviderModelFields
              control={form.control}
              providerName="provider"
              modelName="model"
              optionsName="model_options"
              computerId={computerId}
              description="Unless a project picks one"
              onPick={(provider, model, options) => {
                form.setValue("provider", provider, { shouldDirty: true });
                form.setValue("model", model, { shouldDirty: true });
                form.setValue("model_options", options, { shouldDirty: true });
              }}
            />
          </div>
          <SettingsRow label="New threads start in" description="Worktrees let runs work side by side.">
            <FormSelect control={form.control} name="start_in" label="New threads start in" hideLabel options={START_IN_OPTIONS} />
          </SettingsRow>
        </SettingsRows>
      </form>
    </SettingsCard>
  );
};
