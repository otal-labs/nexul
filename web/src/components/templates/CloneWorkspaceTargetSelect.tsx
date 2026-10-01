import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { EmptyRow } from "@/components/EmptyRow";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useEditableWorkspaces } from "@/hooks/TemplateHooks";
import { WRITE_PERMISSION, type CloneSource, type CloneTemplateFormData } from "@/models/Template";

interface CloneWorkspaceTargetSelectProps {
  source: CloneSource;
}

// Only workspaces where the viewer may write this kind of template, the source itself left out.
export const CloneWorkspaceTargetSelect = ({ source }: CloneWorkspaceTargetSelectProps) => {
  const { control } = useFormDialogContext<CloneTemplateFormData>();
  const workspaces = useEditableWorkspaces(WRITE_PERMISSION[source.kind]);
  const options = workspaces
    ?.filter((w) => w.id !== source.from.workspace_id)
    .map((w) => ({ value: w.id, label: w.name }));

  return (
    <>
      {!options && <LoadingDisplay />}
      {options && options.length === 0 && <EmptyRow>No other workspace you can edit</EmptyRow>}
      {options && options.length > 0 && <FormSelect control={control} name="target" label="Workspace" options={options} />}
    </>
  );
};
