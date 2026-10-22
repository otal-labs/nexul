import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { ProjectComputersField } from "@/components/settings/ProjectComputersField";
import { RunSettingField } from "@/components/settings/RunSettingField";
import { useModelComputerId } from "@/hooks/PairingHooks";
import { useClearProjectLink, useSetProjectLink } from "@/hooks/PairingProjectHooks";
import {
  linkSaysSomething,
  PLACEMENT_OPTIONS,
  ProjectLinkFormSchema,
  START_IN_OPTIONS,
  type ProjectLink,
  type ProjectLinkFormData,
} from "@/models/Pairing";

const SAME_AS_DEFAULTS = "Same as defaults";

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
  const setLink = useSetProjectLink(projectId, projectName);
  const clearLink = useClearProjectLink(projectId, projectName);
  const form = useForm<ProjectLinkFormData>({ defaultValues: formValues(link), resolver: zodResolver(ProjectLinkFormSchema) });
  const ids = useWatch({ control: form.control, name: "computers" });
  const modelComputer = useModelComputerId(ids);

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
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <div>
          <ProjectComputersField projectId={projectId} />
          {ids.length > 0 && (
            <RunSettingField label="Send runs to">
              <FormSelect control={form.control} name="placement" label="Send runs to" hideLabel placeholder={SAME_AS_DEFAULTS} options={PLACEMENT_OPTIONS} />
            </RunSettingField>
          )}
          <RunSettingField label="Model">
            <HarnessProviderModelFields
              control={form.control}
              providerName="provider"
              modelName="model"
              optionsName="model_options"
              computerId={modelComputer}
              defaultLabel={SAME_AS_DEFAULTS}
              onPick={(provider, model, options) => {
                form.setValue("provider", provider, { shouldDirty: true });
                form.setValue("model", model, { shouldDirty: true });
                form.setValue("model_options", options, { shouldDirty: true });
              }}
            />
          </RunSettingField>
          <RunSettingField label="Threads start in">
            <FormSelect control={form.control} name="start_in" label="Threads start in" hideLabel placeholder={SAME_AS_DEFAULTS} options={START_IN_OPTIONS} />
          </RunSettingField>
        </div>
        <div className="flex items-center gap-2 pt-3">
          {linkSaysSomething(link) && (
            <Button type="button" variant="ghost" size="sm" onClick={onClear} loading={clearLink.isPending}>
              Use defaults
            </Button>
          )}
          <Button type="submit" size="sm" variant={form.formState.isDirty ? "default" : "outline"} className="ml-auto" loading={form.formState.isSubmitting} disabled={!form.formState.isDirty}>
            Save
          </Button>
        </div>
      </form>
    </FormProvider>
  );
};
