import { useId } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useWatch } from "react-hook-form";

import { ActionButton } from "@/components/action/ActionButton";
import { FormInput } from "@/components/FormInput";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useCreateProject } from "@/hooks/ProjectHooks";
import { SaveProjectFormSchema, type SaveProjectFormData } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardProjectStepProps {
  onDone: (projectId: string) => void;
  onBack?: (() => void) | undefined;
}

export const WizardProjectStep = ({ onDone, onBack }: WizardProjectStepProps) => {
  const createProject = useCreateProject();
  const setProjectId = useProjectWizardStore((s) => s.setProjectId);
  const formId = useId();
  const form = useForm<SaveProjectFormData>({
    defaultValues: { name: "", prefix: "", icon: "" },
    resolver: zodResolver(SaveProjectFormSchema),
  });

  const prefix = useWatch({ control: form.control, name: "prefix" }).trim() || "BE";

  const onSubmit = async (data: SaveProjectFormData) => {
    try {
      const project = await createProject.mutateAsync(data);
      setProjectId(project.id, project.name);
      onDone(project.id);
    } catch {
      // Errors surface through the hook's toast; creation is retry-safe.
    }
  };

  return (
    <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="grid grid-cols-[1fr_8rem] items-start gap-4">
      <FormInput
        control={form.control}
        name="name"
        label="Project name"
        placeholder="e.g. Backend platform"
        autoFocus
      />
      <FormInput
        control={form.control}
        name="prefix"
        label="Prefix"
        placeholder="e.g. BE"
        className="font-mono placeholder:font-sans"
        maxLength={5}
        transform={(value) => value.toUpperCase()}
      />
      <dl className="col-span-2 mt-2 grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 border-t border-border pt-5 text-xs">
        <dt className="text-muted-foreground">Ticket keys</dt>
        <dd className="font-mono text-foreground">
          {prefix}-1, {prefix}-2, {prefix}-3
        </dd>
        <dt className="text-muted-foreground">Next</dt>
        <dd className="text-pretty text-muted-foreground">Who can open it, the repository it builds from, and how it deploys.</dd>
      </dl>
      <WizardFooter onBack={onBack}>
        <ActionButton permission="projects:create" type="submit" form={formId} loading={form.formState.isSubmitting}>
          Continue
        </ActionButton>
      </WizardFooter>
    </form>
  );
};
