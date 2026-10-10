import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormInput } from "@/components/FormInput";
import { UnfinishedProjectBanner } from "@/components/project/UnfinishedProjectBanner";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useCreateProject } from "@/hooks/ProjectHooks";
import { SaveProjectFormSchema, type SaveProjectFormData } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";
import { useUnfinishedProjectStore } from "@/stores/unfinishedProjectStore";

interface WizardProjectStepProps {
  onDone: () => void;
  onBack?: (() => void) | undefined;
  continueLabel?: string;
}

export const WizardProjectStep = ({ onDone, onBack, continueLabel = "Continue" }: WizardProjectStepProps) => {
  const createProject = useCreateProject();
  const setProjectId = useProjectWizardStore((s) => s.setProjectId);
  const startUnfinished = useUnfinishedProjectStore((s) => s.start);
  const form = useForm<SaveProjectFormData>({
    defaultValues: { name: "", prefix: "", icon: "" },
    resolver: zodResolver(SaveProjectFormSchema),
  });

  const onSubmit = async (data: SaveProjectFormData) => {
    try {
      const project = await createProject.mutateAsync(data);
      setProjectId(project.id, project.name);
      startUnfinished(project.id);
      onDone();
    } catch {
      // Errors surface through the hook's toast; creation is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <UnfinishedProjectBanner />
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
        maxLength={5}
        transform={(value) => value.toUpperCase()}
      />
      <WizardFooter onBack={onBack}>
        <Button type="submit" loading={form.formState.isSubmitting}>
          {continueLabel}
        </Button>
      </WizardFooter>
    </form>
  );
};
