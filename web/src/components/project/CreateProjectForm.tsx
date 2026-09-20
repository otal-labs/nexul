import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateProject } from "@/hooks/ProjectHooks";
import type { SaveProjectFormData } from "@/models/Project";

export const CreateProjectForm = () => {
  const { control, onSubmit } = useFormDialogContext<SaveProjectFormData>();
  const createProject = useCreateProject();

  onSubmit(async ({ name, prefix, icon }) => {
    const project = await createProject.mutateAsync({ name, prefix, icon });
    return { name: project.name, prefix: project.prefix, icon };
  });

  return (
    <>
      <FormInput
        control={control}
        name="name"
        label="Project name"
        placeholder="e.g. Backend platform"
        autoFocus
      />
      <FormInput
        control={control}
        name="prefix"
        label="Prefix"
        placeholder="e.g. BE"
        maxLength={5}
      />
    </>
  );
};
