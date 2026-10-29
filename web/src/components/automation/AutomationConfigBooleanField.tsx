import { Controller, type Control } from "react-hook-form";

import { Switch } from "@/components/ui/switch";
import type { AutomationConfigField } from "@/models/Automation";

interface AutomationConfigBooleanFieldProps {
  control: Control<Record<string, string>>;
  field: AutomationConfigField;
}

// The form keeps every value as text, so a switch stores "true" or "false" and configValuesToSave sends the boolean.
export const AutomationConfigBooleanField = ({ control, field }: AutomationConfigBooleanFieldProps) => (
  <Controller
    control={control}
    name={field.key}
    render={({ field: rhf }) => (
      <Switch
        id={field.key}
        checked={rhf.value === "true"}
        onCheckedChange={(checked) => rhf.onChange(String(checked))}
        disabled={rhf.disabled ?? false}
        className="block"
      />
    )}
  />
);
