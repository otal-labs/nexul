import { zodResolver } from "@hookform/resolvers/zod";
import { CheckIcon } from "lucide-react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { RunSettingField } from "@/components/settings/RunSettingField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useModelComputerId, useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import { useFlash } from "@/hooks/useFlash";
import {
  PairingDefaultsFormSchema,
  PLACEMENT_OPTIONS,
  START_IN_OPTIONS,
  StartIn,
  type PairingDefaults,
  type PairingDefaultsFormData,
} from "@/models/Pairing";

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
}

// Split out so defaultValues only seed from a loaded `defaults`, avoiding the F2 "= [] trap".
export const PairingDefaultsForm = ({ defaults }: PairingDefaultsFormProps) => {
  const update = useUpdatePairingDefaults();
  const modelComputer = useModelComputerId();
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
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <div>
        <RunSettingField label="Send runs to">
          <FormSelect control={form.control} name="placement" label="Send runs to" hideLabel options={PLACEMENT_OPTIONS} />
        </RunSettingField>
        <RunSettingField label="Model">
          <HarnessProviderModelFields
            control={form.control}
            providerName="provider"
            modelName="model"
            optionsName="model_options"
            computerId={modelComputer}
            onPick={(provider, model, options) => {
              form.setValue("provider", provider, { shouldDirty: true });
              form.setValue("model", model, { shouldDirty: true });
              form.setValue("model_options", options, { shouldDirty: true });
            }}
          />
        </RunSettingField>
        <RunSettingField label="Threads start in">
          <FormSelect control={form.control} name="start_in" label="Threads start in" hideLabel options={START_IN_OPTIONS} />
        </RunSettingField>
      </div>
      <div className="flex justify-end pt-3">
        <Button type="submit" size="sm" variant={form.formState.isDirty ? "default" : "outline"} loading={form.formState.isSubmitting} disabled={!form.formState.isDirty && !saved}>
          {saved && <CheckIcon className="size-4" aria-hidden />}
          {saved ? "Saved" : "Save"}
        </Button>
      </div>
    </form>
  );
};
