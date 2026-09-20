import { ChevronDown } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";

import { HarnessComputerField } from "@/components/settings/HarnessComputerField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useFetchHarnessProviders, useFetchPresence, useListComputers } from "@/hooks/PairingHooks";

export interface HarnessPick {
  computer_id: string;
  provider: string;
  model: string;
}

interface HarnessPickerPillProps {
  value: HarnessPick;
  onChange: (next: HarnessPick) => void;
}

// The popover's own form: a subscription (allowed under F5) forwards every change to the caller immediately,
// so the pill's label and the run's payload never fall out of sync with what's shown.
const HarnessPickerForm = ({ value, onChange }: HarnessPickerPillProps) => {
  const { data: computers } = useListComputers();
  const { data: presence } = useFetchPresence();
  const form = useForm<HarnessPick>({ defaultValues: value });

  useEffect(() => {
    const subscription = form.watch((watched) => onChange(watched as HarnessPick));
    return () => subscription.unsubscribe();
  }, [form, onChange]);

  return (
    <div className="space-y-3">
      {(!computers || computers.length === 0) && (
        <p className="text-xs text-muted-foreground">Pair a harness in Settings to run plays.</p>
      )}
      {computers && computers.length > 0 && (
        <HarnessComputerField
          control={form.control}
          name="computer_id"
          computers={computers}
          presence={presence ?? {}}
          onChangeValue={() => {
            form.setValue("provider", "");
            form.setValue("model", "");
          }}
        />
      )}
      {computers && computers.length > 0 && (
        <HarnessProviderModelFields
          control={form.control}
          providerName="provider"
          modelName="model"
          computerId={form.watch("computer_id")}
          setModel={(v) => form.setValue("model", v)}
        />
      )}
    </div>
  );
};

// The dialog footer's harness picker: a mono pill naming the computer, provider, and model, opening a
// popover with the same fields the pairing settings use to change any of the three for this run only.
export const HarnessPickerPill = ({ value, onChange }: HarnessPickerPillProps) => {
  const { data: computers } = useListComputers();
  const { data: providers } = useFetchHarnessProviders(value.computer_id);
  const [open, setOpen] = useState(false);

  const computerName = computers?.find((c) => c.id === value.computer_id)?.name;
  const providerEntry = providers?.find((p) => p.id === value.provider);
  const providerLabel = value.provider === "" ? "Provider default" : (providerEntry?.name ?? value.provider);
  const modelLabel = value.model === "" ? "Model default" : (providerEntry?.models.find((m) => m.slug === value.model)?.name ?? value.model);
  const label = computerName ? `${computerName} · ${providerLabel} · ${modelLabel}` : "Pick a harness";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="inline-flex items-center gap-1.5 rounded-full border border-border px-3 py-1 font-mono text-[11px] text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/40"
        >
          {label}
          <ChevronDown className="size-3" aria-hidden />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80">
        <HarnessPickerForm value={value} onChange={onChange} />
      </PopoverContent>
    </Popover>
  );
};
