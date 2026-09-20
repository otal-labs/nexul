import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormSelect } from "@/components/ticket/FormSelect";
import { HarnessProjectField } from "@/components/settings/HarnessProjectField";
import { HarnessProviderModelFields } from "@/components/settings/HarnessProviderModelFields";
import { useClearProjectLink, useSetProjectLink } from "@/hooks/PairingHooks";
import { ProjectLinkFormSchema, type Computer, type ProjectLink, type ProjectLinkFormData } from "@/models/Pairing";

interface ProjectLinkFormProps {
  projectId: string;
  link: ProjectLink;
  computers: Computer[];
}

// Split out so defaultValues only seed from a loaded `link` — same F2 "= [] trap" fix.
export const ProjectLinkForm = ({ projectId, link, computers }: ProjectLinkFormProps) => {
  const setLink = useSetProjectLink(projectId);
  const clearLink = useClearProjectLink(projectId);
  const isLinked = !!link.computer_id;

  const form = useForm<ProjectLinkFormData>({
    defaultValues: {
      computer_id: link.computer_id ?? "",
      harness_project_id: link.harness_project_id ?? "",
      provider: link.provider ?? "",
      model: link.model ?? "",
    },
    resolver: zodResolver(ProjectLinkFormSchema),
  });

  const onSubmit = async (data: ProjectLinkFormData) => {
    try {
      const saved = await setLink.mutateAsync(data);
      form.reset({
        computer_id: saved.computer_id ?? "",
        harness_project_id: saved.harness_project_id ?? "",
        provider: saved.provider ?? "",
        model: saved.model ?? "",
      });
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  const onClear = async () => {
    try {
      await clearLink.mutateAsync();
      form.reset({ computer_id: "", harness_project_id: "", provider: "", model: "" });
    } catch {
      // Error is surfaced by the hook's toast.
    }
  };

  if (computers.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Pair a computer in your pairing settings first, then come back to link one to this project.
      </p>
    );
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <FormSelect
        control={form.control}
        name="computer_id"
        label="Computer"
        placeholder="Not linked"
        options={computers.map((c) => ({ value: c.id, label: c.name }))}
      />
      <HarnessProjectField
        control={form.control}
        name="harness_project_id"
        label="T3 project"
        computerId={form.watch("computer_id")}
      />
      <HarnessProviderModelFields
        control={form.control}
        providerName="provider"
        modelName="model"
        computerId={form.watch("computer_id")}
        setModel={(value) => form.setValue("model", value)}
      />
      <div className="flex flex-wrap gap-2">
        <Button type="submit" disabled={form.formState.isSubmitting}>
          {form.formState.isSubmitting ? "Saving…" : "Save link"}
        </Button>
        {isLinked && (
          <Button type="button" variant="outline" onClick={onClear} disabled={clearLink.isPending}>
            {clearLink.isPending ? "Clearing…" : "Clear link"}
          </Button>
        )}
      </div>
    </form>
  );
};
