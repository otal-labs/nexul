import { useController, type Control, type FieldValues, type Path } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { CloneRepositoriesSection } from "@/components/settings/CloneRepositoriesSection";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchHarnessProjects } from "@/hooks/PairingHooks";

interface HarnessProjectFieldProps<T extends FieldValues> {
  control: Control<T>;
  name: Path<T>;
  label: string;
  computerId: string;
  // The Nexul project whose repositories missing from the computer are offered to clone there.
  projectId?: string;
}

// Fed by the computer's live project registry; any failure degrades to manual id input.
export const HarnessProjectField = <T extends FieldValues>({ control, name, label, computerId, projectId }: HarnessProjectFieldProps<T>) => {
  const projects = useFetchHarnessProjects(computerId);
  const { field, fieldState } = useController({ control, name });

  // The manual input would show the saved raw id until the list arrives, so loading shows the label and loader alone.
  if (computerId && projects.isPending) {
    return (
      <div className="space-y-2">
        <p className="text-sm font-medium">{label}</p>
        <LoadingDisplay label="Loading projects" className="justify-start p-0" />
      </div>
    );
  }

  const listed = !!projects.data && projects.data.length > 0;
  // A saved id means nothing to a person, so it stays hidden until Change clears it for a new one.
  const savedOffline = projects.isError && !!field.value && !fieldState.isDirty;

  return (
    <div className="space-y-2">
      {listed && (
        <FormCombobox
          control={control}
          name={name}
          label={label}
          placeholder="Pick a project"
          options={projects.data.map((p) => ({ value: p.id, label: p.title }))}
        />
      )}
      {savedOffline && (
        <div className="space-y-2">
          <p className="text-sm font-medium">{label}</p>
          <div className="flex items-center gap-2">
            <p className="flex h-9 min-w-0 flex-1 items-center rounded-md border border-input bg-muted/40 px-3 text-sm text-muted-foreground">Saved project</p>
            <Button type="button" variant="outline" onClick={() => field.onChange("")}>
              Change
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">Its name shows when this computer is back online.</p>
        </div>
      )}
      {!listed && !savedOffline && (
        <div className="space-y-1">
          <FormInput control={control} name={name} label={label} placeholder="e.g. proj_abc123" />
          {!computerId && <p className="text-xs text-muted-foreground">Pick a computer to load its projects.</p>}
          {projects.isError && (
            <p className="text-xs text-muted-foreground">
              Couldn't load this computer's projects. Check it's online, or paste the project id.
            </p>
          )}
          {projects.data?.length === 0 && (
            <p className="text-xs text-muted-foreground">This computer has no T3 projects yet.</p>
          )}
        </div>
      )}
      {projectId && projects.data && (
        <CloneRepositoriesSection projectId={projectId} computerId={computerId} projects={projects.data} onCloned={field.onChange} />
      )}
    </div>
  );
};
