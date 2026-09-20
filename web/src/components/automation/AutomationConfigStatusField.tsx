import { Controller, type Control } from "react-hook-form";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchProjectStatuses } from "@/hooks/StatusHooks";
import type { AutomationConfigField } from "@/models/Automation";

interface AutomationConfigStatusFieldProps {
  control: Control<Record<string, string>>;
  field: AutomationConfigField;
}

// Dropdown of the target project's status columns; statuses are project-scoped, so field.project_id picks which.
export const AutomationConfigStatusField = ({ control, field }: AutomationConfigStatusFieldProps) => {
  const { data: statuses } = useFetchProjectStatuses(field.project_id);

  return (
    <Controller
      control={control}
      name={field.key}
      render={({ field: rhf }) => (
        <Select value={rhf.value} onValueChange={rhf.onChange} disabled={!field.project_id}>
          <SelectTrigger id={field.key}>
            <SelectValue placeholder={field.project_id ? "Select a status" : "No project declared for this field"} />
          </SelectTrigger>
          <SelectContent>
            {(statuses ?? []).map((status) => (
              <SelectItem key={status.id} value={status.id}>
                {status.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    />
  );
};
