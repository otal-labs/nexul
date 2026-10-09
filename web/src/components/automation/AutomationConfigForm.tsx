import { useForm } from "react-hook-form";

import { AutomationConfigFieldControl } from "@/components/automation/AutomationConfigFieldControl";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SaveButton } from "@/components/SaveButton";
import { useUpdateAutomationConfig } from "@/hooks/AutomationHooks";
import { parseConfigSchema, parseConfigValues } from "@/models/Automation";
import type { Automation } from "@/models/Automation";

interface AutomationConfigFormProps {
  automation: Automation;
}

// Renders a real form from the code-declared config schema — never a
// generic JSON editor. Values round-trip through PATCH /config.
export const AutomationConfigForm = ({ automation }: AutomationConfigFormProps) => {
  const { fields } = parseConfigSchema(automation.config_schema);
  const values = parseConfigValues(automation.config_values);
  const updateConfig = useUpdateAutomationConfig(automation.id);

  const form = useForm<Record<string, string>>({
    defaultValues: Object.fromEntries(fields.map((f) => [f.key, values[f.key] ?? f.default ?? ""])),
  });

  const onSubmit = (data: Record<string, string>) => updateConfig.mutate(data);

  return (
    <SettingsCard
      id="configuration"
      title="Configuration"
      description="Declared by the automation's code. The next run uses the saved values."
      footer={
        fields.length > 0 && (
          <SaveButton
            type="submit"
            form="automation-config"
            size="sm"
            loading={updateConfig.isPending}
            savedAt={updateConfig.isSuccess ? updateConfig.submittedAt : undefined}
          >
            Save configuration
          </SaveButton>
        )
      }
    >
      {fields.length === 0 && <EmptyRow>This automation has no settings.</EmptyRow>}
      {fields.length > 0 && (
        <form id="automation-config" onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          {fields.map((field) => (
            <AutomationConfigFieldControl key={field.key} control={form.control} field={field} />
          ))}
        </form>
      )}
    </SettingsCard>
  );
};
