import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useUpdatePairingDefaults } from "@/hooks/PairingHooks";
import { PairingDefaultsFormSchema, type Computer, type PairingDefaults, type PairingDefaultsFormData } from "@/models/Pairing";

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
    },
    resolver: zodResolver(PairingDefaultsFormSchema),
  });

  const onSubmit = async (data: PairingDefaultsFormData) => {
    try {
      const saved = await update.mutateAsync(data);
      form.reset({
        default_computer_id: saved.default_computer_id ?? "",
        fallback_project_id: saved.fallback_project_id ?? "",
        provider: saved.provider ?? "",
        model: saved.model ?? "",
      });
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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
        computerId={form.watch("default_computer_id")}
      />
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        computerId={form.watch("default_computer_id")}
        setModel={(value) => form.setValue("model", value)}
      />
      <Button type="submit" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting ? "Saving…" : "Save defaults"}
      </Button>
    </form>
  );
};
