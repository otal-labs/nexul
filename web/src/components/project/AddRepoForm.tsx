import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useAddProjectRepo } from "@/hooks/ProjectHooks";
import type { AddProjectRepoFormData } from "@/models/Project";

interface AddRepoFormProps {
  projectId: string;
}

// Only "github" is a real connector today; a single-option select leaves room to grow into a picker later.
const CONNECTOR_OPTIONS = [{ value: "github", label: "GitHub" }];

export const AddRepoForm = ({ projectId }: AddRepoFormProps) => {
  const { control, onSubmit } = useFormDialogContext<AddProjectRepoFormData>();
  const addRepo = useAddProjectRepo();

  onSubmit(async ({ owner, name, connectorId }) => {
    await addRepo.mutateAsync({ projectId, owner, name, connectorId });
    return { owner, name, connectorId };
  });

  return (
    <div className="space-y-4">
      <FormInput control={control} name="owner" label="Repository owner" placeholder="owner" autoFocus />
      <FormInput control={control} name="name" label="Repository name" placeholder="repo name" />
      <FormSelect control={control} name="connectorId" label="Connector" options={CONNECTOR_OPTIONS} />
    </div>
  );
};
