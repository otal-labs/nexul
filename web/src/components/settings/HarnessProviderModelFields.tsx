import { useWatch, type Control, type FieldValues, type Path, type PathValue } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { FormInput } from "@/components/FormInput";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";

interface HarnessProviderModelFieldsProps<T extends FieldValues> {
  control: Control<T>;
  providerName: Path<T>;
  modelName: Path<T>;
  computerId: string;
  /** Clears the model on provider change — a model slug only means something within its provider. */
  setModel: (value: PathValue<T, Path<T>>) => void;
}

// Fed by the computer's live provider registry; empty means "T3's default"; a needs-setup provider stays pickable, the run refuses it.
export const HarnessProviderModelFields = <T extends FieldValues>({
  control,
  providerName,
  modelName,
  computerId,
  setModel,
}: HarnessProviderModelFieldsProps<T>) => {
  const providers = useFetchHarnessProviders(computerId);
  const providerValue = useWatch({ control, name: providerName }) as string;
  const selected = providers.data?.find((p) => p.id === providerValue);

  if (providers.data && providers.data.length > 0) {
    return (
      <div className="grid gap-4 sm:grid-cols-2">
        <FormCombobox
          control={control}
          name={providerName}
          label="Provider"
          placeholder="Computer default"
          options={providers.data.map((p) => ({ value: p.id, label: p.name, ...(p.needs_setup && { hint: "needs setup" }) }))}
          onChangeValue={() => setModel("" as PathValue<T, Path<T>>)}
        />
        {selected ? (
          <FormCombobox
            control={control}
            name={modelName}
            label="Model"
            placeholder="Provider default"
            options={selected.models.map((m) => ({ value: m.slug, label: m.name }))}
          />
        ) : (
          <FormInput control={control} name={modelName} label="Model" placeholder="Provider default" />
        )}
      </div>
    );
  }

  return (
    <div className="space-y-1">
      <div className="grid gap-4 sm:grid-cols-2">
        <FormInput control={control} name={providerName} label="Provider" placeholder="e.g. claude, opencode" />
        <FormInput control={control} name={modelName} label="Model" placeholder="e.g. claude-sonnet-4-5" />
      </div>
      {providers.isPending && !!computerId && <p className="text-xs text-muted-foreground">Loading providers…</p>}
      {providers.isError && (
        <p className="text-xs text-muted-foreground">
          Couldn't load providers from this computer — is it online? Type them manually.
        </p>
      )}
    </div>
  );
};
