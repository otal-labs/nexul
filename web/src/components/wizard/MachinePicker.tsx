import type { Control, FieldValues, Path } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { useFetchMachines } from "@/hooks/MachineHooks";

interface MachinePickerProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
}

// A stack targets a machine by name, the same way RunnerPicker targets a runner by name (issue 05: "the
// wizard's target step lists machines, not runners").
export const MachinePicker = <T extends FieldValues>({ control, name }: MachinePickerProps<T>) => {
  const { data: machines } = useFetchMachines();
  const options = (machines ?? []).map((machine) => ({ value: machine.name, label: machine.name }));

  return (
    <div className="space-y-2">
      <FormCombobox control={control} name={name} label="Machine" options={options} placeholder="Choose a machine…" />
      {machines && machines.length === 0 && (
        <p className="text-xs text-muted-foreground">No machines yet — connect a runner first.</p>
      )}
    </div>
  );
};
