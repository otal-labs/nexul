import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useFetchCloneDestinations } from "@/hooks/MemoryHooks";
import { useEditableWorkspaces } from "@/hooks/TemplateHooks";
import { WRITE_PERMISSION, type CloneSource, type CloneTemplateFormData } from "@/models/Template";

interface CloneProjectTargetSelectProps {
  source: CloneSource;
}

// Every project in a workspace where the viewer may edit projects, labelled with its workspace, the source left out.
export const CloneProjectTargetSelect = ({ source }: CloneProjectTargetSelectProps) => {
  const { control } = useFormDialogContext<CloneTemplateFormData>();
  const { data: destinations, error } = useFetchCloneDestinations();
  const editable = useEditableWorkspaces(WRITE_PERMISSION[source.kind]);
  const options =
    destinations &&
    editable &&
    destinations
      .filter((d) => editable.some((w) => w.id === d.workspace.id))
      .flatMap((d) => d.projects.map((p) => ({ value: p.id, label: `${d.workspace.name} / ${p.name}` })))
      .filter((option) => option.value !== source.from.project_id);

  return (
    <>
      {!options && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {options && options.length === 0 && <EmptyRow>No other project you can edit</EmptyRow>}
      {options && options.length > 0 && <FormSelect control={control} name="target" label="Project" options={options} />}
    </>
  );
};
