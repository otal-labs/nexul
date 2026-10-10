import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";

import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchHarnessProviders } from "@/hooks/PairingHooks";
import { findModel, optionsLabel } from "@/models/ModelPick";
import type { OptionSetting } from "@/models/Pairing";

export interface HarnessPick {
  computer_id: string;
  provider: string;
  model: string;
  model_options: OptionSetting[];
}

interface HarnessPickerPillProps {
  value: HarnessPick;
  onChange: (next: HarnessPick) => void;
}

// The popover's own form: a subscription (allowed under F5) forwards every change to the caller immediately,
// so the pill's label and the run's payload never fall out of sync with what's shown.
const HarnessPickerForm = ({ value, onChange }: HarnessPickerPillProps) => {
  const form = useForm<HarnessPick>({ defaultValues: value });
  const computerId = useWatch({ control: form.control, name: "computer_id" });

  useEffect(() => form.subscribe({ formState: { values: true }, callback: ({ values }) => onChange(values) }), [form, onChange]);

  return (
    <HarnessProviderModelFields
      control={form.control}
      providerName="provider"
      modelName="model"
      optionsName="model_options"
      computerId={computerId}
      description="For this run only"
      onPick={(provider, model, options) => {
        form.setValue("provider", provider);
        form.setValue("model", model);
        form.setValue("model_options", options);
      }}
    />
  );
};

// The dialog footer's model picker: a mono pill naming the provider and model on the run's computer, opening a
// popover with the same fields the pairing settings use to change either for this run only.
export const HarnessPickerPill = ({ value, onChange }: HarnessPickerPillProps) => {
  const { data: providers } = useFetchHarnessProviders(value.computer_id);
  const [open, setOpen] = useState(false);

  const providerEntry = providers?.find((p) => p.id === value.provider);
  const providerLabel = value.provider === "" ? "Provider default" : (providerEntry?.name ?? value.provider);
  const model = findModel(providers ?? [], value);
  const modelLabel = value.model === "" ? "Model default" : (model?.name ?? value.model);
  const optionsText = model?.options && value.model_options.length > 0 ? ` · ${optionsLabel(model.options, value.model_options)}` : "";
  const label = `${providerLabel} · ${modelLabel}${optionsText}`;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 rounded-full border border-border px-3 py-1 font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40"
        >
          {label}
          <ChevronDown className="size-3" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-96">
        <HarnessPickerForm value={value} onChange={onChange} />
      </PopoverContent>
    </Popover>
  );
};
