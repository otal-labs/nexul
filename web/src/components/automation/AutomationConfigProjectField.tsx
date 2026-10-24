import { Controller, type Control } from "react-hook-form";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { AutomationConfigField } from "@/models/Automation";

interface AutomationConfigProjectFieldProps {
  control: Control<Record<string, string>>;
  field: AutomationConfigField;
}

// Dropdown of the selected workspace's projects for a "project" config knob.
export const AutomationConfigProjectField = ({ control, field }: AutomationConfigProjectFieldProps) => {
  const { data: projects } = useFetchProjects();

  return (
    <Controller
      control={control}
      name={field.key}
      render={({ field: rhf }) => (
        <Select value={rhf.value} onValueChange={rhf.onChange}>
          <SelectTrigger id={field.key}>
            <SelectValue placeholder="Select a project" />
          </SelectTrigger>
          <SelectContent>
            {(projects ?? []).map((project) => (
              <SelectItem key={project.id} value={project.id}>
                {project.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
    />
  );
};
