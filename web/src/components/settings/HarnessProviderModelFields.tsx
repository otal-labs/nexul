import { useWatch, type Control, type FieldValues, type Path } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { ModelOptionsPicker } from "@/components/model/ModelOptionsPicker";
import { ModelPicker } from "@/components/model/ModelPicker";
import { ModelSettingRow } from "@/components/model/ModelSettingRow";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { findModel } from "@/models/ModelPick";
import type { OptionSetting } from "@/models/Pairing";

interface HarnessProviderModelFieldsProps<T extends FieldValues> {
  control: Control<T>;
  providerName: Path<T>;
  modelName: Path<T>;
  optionsName: Path<T>;
  computerId: string;
  description: string;
  // Options only mean something for their model, so a new model arrives with none set.
  onPick: (provider: string, model: string, options: OptionSetting[]) => void;
}

// Fed by the computer's live provider registry; an empty model means the provider's default, an empty provider the computer's.
export const HarnessProviderModelFields = <T extends FieldValues>({
  control,
  providerName,
  modelName,
  optionsName,
  computerId,
  description,
  onPick,
}: HarnessProviderModelFieldsProps<T>) => {
  const providers = useFetchHarnessProviders(computerId);
  const provider = useWatch({ control, name: providerName }) as string;
  const model = useWatch({ control, name: modelName }) as string;
  const options = (useWatch({ control, name: optionsName }) as OptionSetting[] | undefined) ?? [];

  const loaded = providers.data && providers.data.length > 0 ? providers.data : undefined;

  return (
    <div className="space-y-1">
      {loaded && (
        <ModelSettingRow label="Model" description={description}>
          <ModelPicker
            providers={loaded}
            value={{ provider, model }}
            onChange={(next) => onPick(next.provider, next.model, [])}
            label="Model"
            allowDefault
            className="w-56"
          />
          <ModelOptionsPicker
            options={findModel(loaded, { provider, model })?.options ?? []}
            value={options}
            onChange={(next) => onPick(provider, model, next)}
            label="Model options"
          />
        </ModelSettingRow>
      )}
      {!loaded && (
        <div className="grid gap-4 sm:grid-cols-2">
          <FormInput control={control} name={providerName} label="Provider" placeholder="e.g. claude, opencode" />
          <FormInput control={control} name={modelName} label="Model" placeholder="e.g. claude-sonnet-4-5" />
        </div>
      )}
      {providers.isPending && !!computerId && <p className="text-xs text-muted-foreground">Loading providers…</p>}
      {providers.isError && (
        <p className="text-xs text-muted-foreground">
          Couldn't load providers from this computer — is it online? Type them manually.
        </p>
      )}
    </div>
  );
};
