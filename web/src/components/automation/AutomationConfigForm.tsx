import { useForm } from "react-hook-form";

import { AutomationConfigFieldControl } from "@/components/automation/AutomationConfigFieldControl";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { useUpdateAutomationConfig } from "@/hooks/AutomationHooks";
import { useActionState } from "@/hooks/useActionState";
import { useFlash } from "@/hooks/useFlash";
import { configValuesToSave, parseConfigSchema, parseConfigValues } from "@/models/Automation";
import type { Automation } from "@/models/Automation";

interface AutomationConfigFormProps {
  automation: Automation;
}

// A form built from the schema the code declares, never a JSON editor.
export const AutomationConfigForm = ({ automation }: AutomationConfigFormProps) => {
  const { fields, announced } = parseConfigSchema(automation.config_schema);
  const values = parseConfigValues(automation.config_values);
  const updateConfig = useUpdateAutomationConfig(automation.id);

  const writable = useActionState({ permission: "automations:write" }).state === "live";
  const form = useForm<Record<string, string>>({
    disabled: !writable,
    defaultValues: Object.fromEntries(
      fields.map((f) => [f.key, values[f.key] ?? f.default ?? (f.type === "boolean" ? "false" : "")]),
    ),
  });

  const [saved, flash] = useFlash();
  const onSubmit = (data: Record<string, string>) =>
    updateConfig.mutate(configValuesToSave(fields, data), {
      onSuccess: () => {
        form.reset(data);
        flash();
      },
    });

  return (
    <SettingsCard
      id="configuration"
      title="Configuration"
      footer={
        fields.length > 0 && (
          <GatedSaveBar
            permission="automations:write"
            verb="change automation settings"
            form="automation-config"
            dirty={form.formState.isDirty}
            saving={updateConfig.isPending}
            saved={saved}
            onDiscard={() => form.reset()}
            saveLabel="Save configuration"
          />
        )
      }
    >
      {fields.length === 0 && <EmptyRow flush>{announced ? "Its code declares none." : "Announced when its code first connects."}</EmptyRow>}
      {fields.length > 0 && (
        <form id="automation-config" onSubmit={form.handleSubmit(onSubmit)} className="max-w-[36rem] space-y-4">
          {fields.map((field) => (
            <AutomationConfigFieldControl key={field.key} control={form.control} field={field} />
          ))}
        </form>
      )}
    </SettingsCard>
  );
};
