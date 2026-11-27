import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { ProjectPlaysPlacement } from "@/components/agent/ProjectPlaysPlacement";
import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { ProjectComputersField } from "@/components/settings/ProjectComputersField";
import { RunSettingField } from "@/components/settings/RunSettingField";
import { useModelComputerId } from "@/hooks/PairingHooks";
import { useClearProjectLink, useSetProjectLink } from "@/hooks/PairingProjectHooks";
import { useAutoSaveForm } from "@/hooks/useAutoSaveForm";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import {
  linkSaysSomething,
  PLACEMENT_OPTIONS,
  projectLinkFormValues,
  ProjectLinkFormSchema,
  START_IN_OPTIONS,
  type ProjectLink,
  type ProjectLinkFormData,
} from "@/models/Pairing";

const SAME_AS_DEFAULTS = "Same as defaults";

interface ProjectLinkFormProps {
  projectId: string;
  projectName: string;
  link: ProjectLink;
  onSaved?: () => void;
}

// Split out so defaultValues only seed from a loaded `link`, the same F2 "= [] trap" fix. Every change writes at once.
export const ProjectLinkForm = ({ projectId, projectName, link, onSaved }: ProjectLinkFormProps) => {
  const setLink = useSetProjectLink(projectId, projectName);
  const clearLink = useClearProjectLink(projectId, projectName);
  const { open: confirm } = useConfirmationDialog();
  const form = useForm<ProjectLinkFormData>({ defaultValues: projectLinkFormValues(link), resolver: zodResolver(ProjectLinkFormSchema) });
  const ids = useWatch({ control: form.control, name: "computers" });
  const modelComputer = useModelComputerId(ids);
  useAutoSaveForm(form, async (values) => projectLinkFormValues(await setLink.mutateAsync(values)), onSaved);

  const onReset = async () => {
    const ok = await confirm({
      title: `Reset ${projectName} to defaults?`,
      message: "Its own computers, model and start folder are cleared, so its runs follow your defaults.",
      confirmLabel: "Reset to defaults",
    });
    if (!ok) return;
    try {
      await clearLink.mutateAsync();
      form.reset(projectLinkFormValues({ project_id: projectId, computers: [], checkouts: form.getValues("checkouts") }));
      onSaved?.();
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <FormProvider {...form}>
      <div className="space-y-3">
        <div className="@container divide-y divide-border rounded-lg border border-border px-4">
          <ProjectComputersField projectId={projectId} projectName={projectName} />
          {ids.length > 0 && (
            <RunSettingField label="If several are free">
              <FormSelect control={form.control} name="placement" label="If several are free" hideLabel placeholder={SAME_AS_DEFAULTS} options={PLACEMENT_OPTIONS} />
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
                form.setValue("provider", provider);
                form.setValue("model", model);
                form.setValue("model_options", options);
              }}
            />
          </RunSettingField>
          <RunSettingField label="Threads start in">
            <FormSelect control={form.control} name="start_in" label="Threads start in" hideLabel placeholder={SAME_AS_DEFAULTS} options={START_IN_OPTIONS} />
          </RunSettingField>
          <ProjectPlaysPlacement projectId={projectId} />
        </div>
        <Button type="button" variant="outline" size="sm" onClick={() => void onReset()} loading={clearLink.isPending} disabled={!linkSaysSomething(link)}>
          Reset to defaults
        </Button>
      </div>
    </FormProvider>
  );
};
