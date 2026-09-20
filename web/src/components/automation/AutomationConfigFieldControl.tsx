import type { Control } from "react-hook-form";

import { AutomationConfigChannelField } from "@/components/automation/AutomationConfigChannelField";
import { AutomationConfigStatusField } from "@/components/automation/AutomationConfigStatusField";
import { FormInput } from "@/components/FormInput";
import type { AutomationConfigField } from "@/models/Automation";

interface AutomationConfigFieldControlProps {
  control: Control<Record<string, string>>;
  field: AutomationConfigField;
}

// Maps a declared config field's type to a real control, one shared label/row shape for all types.
export const AutomationConfigFieldControl = ({ control, field }: AutomationConfigFieldControlProps) => (
  <div className="space-y-2">
    <label htmlFor={field.key} className="text-sm font-medium">
      {field.label}
      {field.required && <span className="text-destructive"> *</span>}
    </label>
    {field.description && <p className="text-xs text-muted-foreground">{field.description}</p>}
    {field.type === "status" && <AutomationConfigStatusField control={control} field={field} />}
    {field.type === "channel" && <AutomationConfigChannelField control={control} field={field} />}
    {field.type === "string" && (
      <FormInput control={control} name={field.key} label={field.label} hideLabel id={field.key} />
    )}
  </div>
);
