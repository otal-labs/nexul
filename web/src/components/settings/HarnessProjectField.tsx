import type { Control, FieldValues, Path } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { FormInput } from "@/components/FormInput";
import { useFetchHarnessProjects } from "@/hooks/PairingHooks";

interface HarnessProjectFieldProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  computerId: string;
}

// Fed by the computer's live project registry; any failure degrades to manual id input.
export const HarnessProjectField = <T extends FieldValues>({ control, name, label, computerId }: HarnessProjectFieldProps<T>) => {
  const projects = useFetchHarnessProjects(computerId);

  if (projects.data && projects.data.length > 0) {
    return (
      <FormCombobox
        control={control}
        name={name}
        label={label}
        placeholder="Pick a project"
        options={projects.data.map((p) => ({ value: p.id, label: p.title }))}
      />
    );
  }

  return (
    <div className="space-y-1">
      <FormInput control={control} name={name} label={label} placeholder="e.g. proj_abc123" />
      {!computerId && <p className="text-xs text-muted-foreground">Pick a computer to load its projects.</p>}
      {projects.isPending && !!computerId && <p className="text-xs text-muted-foreground">Loading projects…</p>}
      {projects.isError && (
        <p className="text-xs text-muted-foreground">
          Couldn't load projects from this computer — is it online? Paste the id manually.
        </p>
      )}
      {projects.data?.length === 0 && (
        <p className="text-xs text-muted-foreground">This computer has no T3 projects yet.</p>
      )}
    </div>
  );
};
