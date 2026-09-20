import { useForm } from "react-hook-form";

import { AutomationConfigFieldControl } from "@/components/automation/AutomationConfigFieldControl";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { Button } from "@/components/ui/button";
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
    <section className="space-y-4 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold">Configuration</h2>
      {fields.length === 0 && <NoDataDisplay message="This automation has no config knobs" size="compact" />}
      {fields.length > 0 && (
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          {fields.map((field) => (
            <AutomationConfigFieldControl key={field.key} control={form.control} field={field} />
          ))}
          <Button type="submit" size="sm" disabled={updateConfig.isPending}>
            {updateConfig.isPending ? "Saving…" : "Save configuration"}
          </Button>
        </form>
      )}
    </section>
  );
};
