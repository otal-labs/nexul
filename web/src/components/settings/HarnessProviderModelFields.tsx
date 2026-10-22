import { useWatch, type Control, type FieldValues, type Path } from "react-hook-form";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ModelChoice } from "@/components/model/ModelChoice";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { COMPUTER_DEFAULT } from "@/models/ModelPick";
import type { OptionSetting } from "@/models/Pairing";

interface HarnessProviderModelFieldsProps<T extends FieldValues> {
  control: Control<T>;
  providerName: Path<T>;
  modelName: Path<T>;
  optionsName: Path<T>;
  computerId: string;
  // What the empty pick reads as, such as "Same as defaults" inside a project.
  defaultLabel?: string;
  // Options only mean something for their model, so a new model arrives with none set.
  onPick: (provider: string, model: string, options: OptionSetting[]) => void;
}

// Fed by the computer's live provider registry; an empty model means the provider's default, an empty provider the
// computer's. With no computer answering it shows the saved pick and offers no field, because a typed model can't be checked.
export const HarnessProviderModelFields = <T extends FieldValues>({
  control,
  providerName,
  modelName,
  optionsName,
  computerId,
  defaultLabel = COMPUTER_DEFAULT,
  onPick,
}: HarnessProviderModelFieldsProps<T>) => {
  const providers = useFetchHarnessProviders(computerId);
  const provider = useWatch({ control, name: providerName }) as string;
  const model = useWatch({ control, name: modelName }) as string;
  const options = (useWatch({ control, name: optionsName }) as OptionSetting[] | undefined) ?? [];

  const loaded = providers.data && providers.data.length > 0 ? providers.data : undefined;
  const loading = providers.isPending && !!computerId;

  return (
    <div className="min-w-0 flex-1">
      {loading && <LoadingDisplay label="Loading models" className="justify-start p-0" />}
      {loaded && (
        <ModelChoice
          providers={loaded}
          value={{ provider, model }}
          options={options}
          onPick={(next) => onPick(next.provider, next.model, [])}
          onOptions={(next) => onPick(provider, model, next)}
          label="Model"
          allowDefault
          defaultLabel={defaultLabel}
          className="h-9 w-full"
        />
      )}
      {!loaded && !loading && (
        <div className="space-y-1.5">
          <p className="flex h-9 items-center truncate rounded-md border border-input px-3 text-sm text-muted-foreground">
            {model || provider || defaultLabel}
          </p>
          <p className="text-xs text-muted-foreground">Turn on a computer to change it.</p>
        </div>
      )}
    </div>
  );
};
