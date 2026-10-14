import type { Control, FieldValues, Path } from "react-hook-form";

import { FixLink } from "@/components/FixLink";
import { FormCombobox } from "@/components/FormCombobox";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchDeployTargets } from "@/hooks/MachineHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface MachinePickerProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
}

// A service targets a machine by name and any runner on it takes the job; this is the one picker for every target field,
// and it offers only the machines the viewer may deploy to.
export const MachinePicker = <T extends FieldValues>({ control, name }: MachinePickerProps<T>) => {
  const { data: machines } = useFetchDeployTargets();
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  const canOpenRunners = can?.("runners");
  const options = (machines ?? []).map((machine) => ({ value: machine.name, label: machine.name }));

  return (
    <div className="space-y-2">
      <FormCombobox control={control} name={name} label="Machine" options={options} placeholder="Choose a machine…" />
      {machines && machines.length === 0 && (
        <p className="text-xs text-muted-foreground">
          No machine you can deploy to yet.{" "}
          {canOpenRunners && <FixLink to={wsPath("/runners")} label="Add a runner" />}
          {canOpenRunners === false && "Ask an admin for access to a machine."}
        </p>
      )}
    </div>
  );
};
