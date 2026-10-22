import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { ProjectComputersField } from "@/components/settings/ProjectComputersField";
import { useListComputers } from "@/hooks/PairingHooks";
import { useClearProjectLink, useSetProjectLink } from "@/hooks/PairingProjectHooks";
import { linkSaysSomething, ProjectLinkFormSchema, START_IN_OPTIONS, type ProjectLink, type ProjectLinkFormData } from "@/models/Pairing";

const formValues = (link: ProjectLink): ProjectLinkFormData => ({
  provider: link.provider ?? "",
  model: link.model ?? "",
  model_options: link.model_options ?? [],
  start_in: link.start_in ?? "",
  placement: link.placement ?? "",
  computers: link.computers,
  checkouts: link.checkouts,
});

interface ProjectLinkFormProps {
  projectId: string;
  projectName: string;
  link: ProjectLink;
}

// Split out so defaultValues only seed from a loaded `link`, the same F2 "= [] trap" fix.
export const ProjectLinkForm = ({ projectId, projectName, link }: ProjectLinkFormProps) => {
  const { data: computers } = useListComputers();
  const setLink = useSetProjectLink(projectId, projectName);
  const clearLink = useClearProjectLink(projectId, projectName);
  const form = useForm<ProjectLinkFormData>({ defaultValues: formValues(link), resolver: zodResolver(ProjectLinkFormSchema) });
  const ids = useWatch({ control: form.control, name: "computers" });
  const modelComputer = ids[0] ?? computers?.[0]?.id ?? "";

  const onSubmit = async (data: ProjectLinkFormData) => {
    try {
      form.reset(formValues(await setLink.mutateAsync(data)));
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  const onClear = async () => {
    try {
      await clearLink.mutateAsync();
      form.reset(formValues({ project_id: projectId, computers: [], checkouts: form.getValues("checkouts") }));
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
        <ProjectComputersField projectId={projectId} />
        <HarnessProviderModelFields
          control={form.control}
          providerName="provider"
          modelName="model"
          optionsName="model_options"
          computerId={modelComputer}
          description="Model for your agent turns in this project, on every computer"
          onPick={(provider, model, options) => {
            form.setValue("provider", provider, { shouldDirty: true });
            form.setValue("model", model, { shouldDirty: true });
            form.setValue("model_options", options, { shouldDirty: true });
          }}
        />
        <FormSelect control={form.control} name="start_in" label="New threads start in" placeholder="Same as your defaults" options={START_IN_OPTIONS} />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" loading={form.formState.isSubmitting} disabled={!form.formState.isDirty}>
            Save
          </Button>
          {linkSaysSomething(link) && (
            <Button type="button" variant="outline" onClick={onClear} loading={clearLink.isPending}>
              Follow my computers and defaults
            </Button>
          )}
        </div>
      </form>
    </FormProvider>
  );
};
