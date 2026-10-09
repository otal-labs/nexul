import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useClearProjectLink, useSetProjectLink } from "@/hooks/PairingProjectHooks";
import {
  ProjectLinkFormSchema,
  START_IN_OPTIONS,
  type Computer,
  type ProjectLink,
  type ProjectLinkFormData,
} from "@/models/Pairing";

interface ProjectLinkFormProps {
  projectId: string;
  projectName: string;
  link: ProjectLink;
  computers: Computer[];
}

// Split out so defaultValues only seed from a loaded `link` — same F2 "= [] trap" fix.
export const ProjectLinkForm = ({ projectId, projectName, link, computers }: ProjectLinkFormProps) => {
  const setLink = useSetProjectLink(projectId, projectName);
  const clearLink = useClearProjectLink(projectId, projectName);
  const isLinked = !!link.computer_id;

  const form = useForm<ProjectLinkFormData>({
    defaultValues: {
      computer_id: link.computer_id ?? "",
      harness_project_id: link.harness_project_id ?? "",
      provider: link.provider ?? "",
      model: link.model ?? "",
      model_options: link.model_options ?? [],
      start_in: link.start_in ?? "",
    },
    resolver: zodResolver(ProjectLinkFormSchema),
  });
  const computerId = useWatch({ control: form.control, name: "computer_id" });

  const onSubmit = async (data: ProjectLinkFormData) => {
    try {
      const saved = await setLink.mutateAsync(data);
      form.reset({
        computer_id: saved.computer_id ?? "",
        harness_project_id: saved.harness_project_id ?? "",
        provider: saved.provider ?? "",
        model: saved.model ?? "",
        model_options: saved.model_options ?? [],
        start_in: saved.start_in ?? "",
      });
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  const onClear = async () => {
    try {
      await clearLink.mutateAsync();
      form.reset({ computer_id: "", harness_project_id: "", provider: "", model: "", model_options: [], start_in: "" });
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <FormSelect
        control={form.control}
        name="computer_id"
        label="Computer"
        placeholder="Pick a computer"
        options={computers.map((c) => ({ value: c.id, label: c.name }))}
      />
      <HarnessProjectField
        control={form.control}
        name="harness_project_id"
        label="T3 project"
        computerId={computerId}
      />
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        optionsName="model_options"
        computerId={computerId}
        description="Model for your agent turns in this project"
        onPick={(provider, model, options) => {
          form.setValue("provider", provider, { shouldDirty: true });
          form.setValue("model", model, { shouldDirty: true });
          form.setValue("model_options", options, { shouldDirty: true });
        }}
      />
      <FormSelect
        control={form.control}
        name="start_in"
        label="New threads start in"
        placeholder="Same as my defaults"
        options={START_IN_OPTIONS}
      />
      <div className="flex flex-wrap gap-2">
        <Button type="submit" loading={form.formState.isSubmitting}>
          Save
        </Button>
        {isLinked && (
          <Button type="button" variant="outline" onClick={onClear} loading={clearLink.isPending}>
            Use my defaults
          </Button>
        )}
      </div>
    </form>
  );
};
