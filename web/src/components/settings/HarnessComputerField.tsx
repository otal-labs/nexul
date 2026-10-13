import { Controller, type Control, type FieldValues, type Path } from "react-hook-form";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import type { Computer } from "@/models/Pairing";
import { notRunnableYet, sharedComputerName } from "@/models/SharedComputer";

interface HarnessComputerFieldProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
  computers: Computer[];
  presence: Record<string, string>;
  onChangeValue?: (value: string) => void;
  // Lists the computers shared with the viewer too, which can't be picked yet.
  withShared?: boolean;
}

// Every entry is one of the caller's own paired computers; an offline one is disabled with the reason beside it.
export const HarnessComputerField = <T extends FieldValues>({
  control,
  name,
  computers,
  presence,
  onChangeValue,
  withShared = false,
}: HarnessComputerFieldProps<T>) => {
  const { data: shared } = useListSharedComputers();
  const labelId = `${name}-label`;
  return (
    <div className="space-y-2">
      <label id={labelId} className="text-sm font-medium">
        Computer
      </label>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Select
            value={field.value as string}
            onValueChange={(value) => {
              field.onChange(value);
              onChangeValue?.(value);
            }}
          >
            <SelectTrigger id={name} aria-labelledby={labelId}>
              <SelectValue placeholder="Pick a computer" />
            </SelectTrigger>
            <SelectContent>
              {computers.map((computer) => {
                const offline = presence[computer.id] !== "connected";
                return (
                  <SelectItem key={computer.id} value={computer.id} disabled={offline}>
                    {computer.name}
                    {offline && <span className="ml-1.5 text-muted-foreground">Offline</span>}
                  </SelectItem>
                );
              })}
              {withShared &&
                shared?.map((computer) => (
                  <SelectItem key={computer.id} value={computer.id} disabled>
                    {sharedComputerName(computer)}
                    <span className="ml-1.5 text-muted-foreground">{notRunnableYet}</span>
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        )}
      />
    </div>
  );
};
