import { useWatch, type Control, type FieldValues, type Path } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ModelChoice } from "@/components/model/ModelChoice";
import { ModelSettingRow } from "@/components/model/ModelSettingRow";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";
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
          <ModelChoice
            providers={loaded}
            value={{ provider, model }}
            options={options}
            onPick={(next) => onPick(next.provider, next.model, [])}
            onOptions={(next) => onPick(provider, model, next)}
            label="Model"
            allowDefault
            className="w-full md:w-96"
          />
        </ModelSettingRow>
      )}
      {!loaded && (
        <div className="grid gap-4 sm:grid-cols-2">
          <FormInput control={control} name={providerName} label="Provider" placeholder="e.g. claude, opencode" />
          <FormInput control={control} name={modelName} label="Model" placeholder="e.g. claude-sonnet-4-5" />
        </div>
      )}
      {providers.isPending && !!computerId && <LoadingDisplay label="Loading providers" className="justify-start p-0" />}
      {providers.isError && (
        <p className="text-xs text-muted-foreground">
          Couldn't load this computer's providers. Check it's online, or type them in.
        </p>
      )}
    </div>
  );
};
