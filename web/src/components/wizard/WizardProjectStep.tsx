import { useId } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

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
    <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      <FormInput
        control={form.control}
        name="name"
        label="Project name"
        placeholder="e.g. Backend platform"
        className="max-w-80"
        autoFocus
      />
      <FormInput
        control={form.control}
        name="prefix"
        label="Prefix"
        placeholder="e.g. BE"
        className="max-w-80"
        maxLength={5}
        transform={(value) => value.toUpperCase()}
      />
      <WizardFooter onBack={onBack}>
        <ActionButton permission="projects:create" type="submit" form={formId} loading={form.formState.isSubmitting}>
          Continue
        </ActionButton>
      </WizardFooter>
    </form>
  );
};
