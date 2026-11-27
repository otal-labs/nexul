import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { DefaultsOrderList } from "@/components/computers/DefaultsOrderList";
import { FormSelect } from "@/components/ticket/FormSelect";
import { RunSettingField } from "@/components/settings/RunSettingField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useModelComputerId, useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import { useAutoSaveForm } from "@/hooks/useAutoSaveForm";
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
  onSaved?: () => void;
}

// Split out so defaultValues only seed from a loaded `defaults`, avoiding the F2 "= [] trap". Every change writes at once.
export const PairingDefaultsForm = ({ defaults, onSaved }: PairingDefaultsFormProps) => {
  const update = useUpdatePairingDefaults();
  const modelComputer = useModelComputerId();
  const form = useForm<PairingDefaultsFormData>({ defaultValues: formValues(defaults), resolver: zodResolver(PairingDefaultsFormSchema) });
  useAutoSaveForm(form, async (values) => formValues(await update.mutateAsync(values)), onSaved);

  return (
    <div className="@container divide-y divide-border rounded-lg border border-border px-4">
      <RunSettingField label="Computers">
        <DefaultsOrderList />
      </RunSettingField>
      <RunSettingField label="If several are free">
        <FormSelect control={form.control} name="placement" label="If several are free" hideLabel options={PLACEMENT_OPTIONS} />
      </RunSettingField>
      <RunSettingField label="Model">
        <HarnessProviderModelFields
          control={form.control}
          providerName="provider"
          modelName="model"
          optionsName="model_options"
          computerId={modelComputer}
          onPick={(provider, model, options) => {
            form.setValue("provider", provider);
            form.setValue("model", model);
            form.setValue("model_options", options);
          }}
        />
      </RunSettingField>
      <RunSettingField label="Threads start in">
        <FormSelect control={form.control} name="start_in" label="Threads start in" hideLabel options={START_IN_OPTIONS} />
      </RunSettingField>
    </div>
  );
};
