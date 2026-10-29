import { zodResolver } from "@hookform/resolvers/zod";
import type { AxiosError } from "axios";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { FormInput } from "@/components/FormInput";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { WorkspaceUrlField } from "@/components/settings/WorkspaceUrlField";
import { useFlash } from "@/hooks/useFlash";
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
  const [saved, flash] = useFlash();
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
      flash();
    } catch (error) {
      if (refusedSlug(error)) return form.setError("slug", { message: errorMessage(error) });
      toast.error(errorMessage(error));
    }
  };

  return (
    <SettingsCard
      id="workspace"
      title="Workspace"
      footer={
        <SettingsSaveBar
          form="workspace-general"
          dirty={isDirty}
          invalid={!isValid}
          saving={isSubmitting}
          saved={saved}
          onDiscard={() => form.reset()}
        />
      }
    >
      <FormProvider {...form}>
        <form id="workspace-general" onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsRows>
            <SettingsRow label="Name" description="In the sidebar and the switcher." stacked>
              <div className="w-full">
                <FormInput control={form.control} name="name" label="Name" hideLabel className="text-ellipsis" />
              </div>
            </SettingsRow>
            <SettingsRow label="URL" description="Every link uses it. Old links stop working when it changes." stacked>
              <WorkspaceUrlField />
            </SettingsRow>
          </SettingsRows>
        </form>
      </FormProvider>
    </SettingsCard>
  );
};
