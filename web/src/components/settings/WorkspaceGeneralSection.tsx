import { zodResolver } from "@hookform/resolvers/zod";
import type { AxiosError } from "axios";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { FormInput } from "@/components/FormInput";
import { SaveButton } from "@/components/SaveButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { WorkspaceUrlField } from "@/components/settings/WorkspaceUrlField";
import { useUpdateWorkspace } from "@/hooks/WorkspaceHooks";
import { WorkspaceGeneralFormSchema, type Workspace, type WorkspaceGeneralFormData } from "@/models/Workspace";

interface WorkspaceGeneralSectionProps {
  workspace: Workspace;
}

// The server is where a slug's uniqueness is decided, so its refusals of the slug belong on the URL field.
const refusedSlug = (error: unknown) => {
  const status = (error as AxiosError).response?.status;
  return status === 409 || (status === 400 && /slug/i.test(errorMessage(error)));
};

// Gated on workspaces:write by the parent, like every workspace section.
export const WorkspaceGeneralSection = ({ workspace }: WorkspaceGeneralSectionProps) => {
  const updateWorkspace = useUpdateWorkspace();
  const form = useForm<WorkspaceGeneralFormData>({
    mode: "onChange",
    defaultValues: { name: workspace.name, slug: workspace.slug },
    resolver: zodResolver(WorkspaceGeneralFormSchema),
  });
  const { isDirty, isSubmitting, isValid } = form.formState;

  const onSubmit = async (values: WorkspaceGeneralFormData) => {
    const changes = {
      ...(values.name !== workspace.name && { name: values.name }),
      ...(values.slug !== workspace.slug && { slug: values.slug }),
    };
    if (Object.keys(changes).length === 0) return form.reset(values);
    try {
      const saved = await updateWorkspace.mutateAsync({ id: workspace.id, ...changes });
      form.reset({ name: saved.name, slug: saved.slug });
      toast.success("Workspace updated");
    } catch (error) {
      if (refusedSlug(error)) return form.setError("slug", { message: errorMessage(error) });
      toast.error(errorMessage(error));
    }
  };

  return (
    <SettingsCard
      id="workspace"
      title="Workspace"
      description="Its name in the sidebar and the switcher, and its address in every link."
      footer={
        <SaveButton
          type="submit"
          form="workspace-general"
          disabled={!isDirty || !isValid}
          loading={isSubmitting}
          savedAt={updateWorkspace.isSuccess ? updateWorkspace.submittedAt : undefined}
        >
          Save
        </SaveButton>
      }
    >
      <FormProvider {...form}>
        <form id="workspace-general" onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
          <FormInput control={form.control} name="name" label="Name" className="max-w-sm" />
          <WorkspaceUrlField savedSlug={workspace.slug} />
        </form>
      </FormProvider>
    </SettingsCard>
  );
};
