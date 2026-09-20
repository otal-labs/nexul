import { Link } from "react-router";
import type { Control, FieldValues, Path } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { useRunners } from "@/hooks/RunnerHooks";

interface RunnerPickerProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
}

// A service target is the name of the runner that runs it; this is the one picker for every target field.
export const RunnerPicker = <T extends FieldValues>({ control, name }: RunnerPickerProps<T>) => {
  const { data: runners } = useRunners();
  const options = (runners ?? []).map((runner) => ({
    value: runner.name,
    label: runner.connected ? runner.name : `${runner.name} (offline)`,
  }));

  return (
    <div className="space-y-2">
      <FormCombobox control={control} name={name} label="Runs on" options={options} placeholder="Choose a runner…" />
      {runners && runners.length === 0 && (
        <p className="text-xs text-muted-foreground">
          No runners yet.{" "}
          <Link to="/runners" className="underline underline-offset-2 hover:text-foreground">
            Add one from the Runners page.
          </Link>
        </p>
      )}
    </div>
  );
};
