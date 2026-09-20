import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useCreateProject } from "@/hooks/ProjectHooks";
import { SaveProjectFormSchema, type SaveProjectFormData } from "@/models/Project";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardProjectStepProps {
  onDone: () => void;
}

// Same fields as CreateProjectForm (name + prefix); a plain form rather than that dialog-bound component,
// since this rung advances the rail on success instead of closing a dialog.
export const WizardProjectStep = ({ onDone }: WizardProjectStepProps) => {
  const createProject = useCreateProject();
  const setProjectId = useProjectWizardStore((s) => s.setProjectId);
  const form = useForm<SaveProjectFormData>({
    defaultValues: { name: "", prefix: "", icon: "" },
    resolver: zodResolver(SaveProjectFormSchema),
  });

  const onSubmit = async (data: SaveProjectFormData) => {
    try {
      const project = await createProject.mutateAsync(data);
      setProjectId(project.id, project.name);
      onDone();
    } catch {
      // Errors surface through the hook's toast; creation is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <FormInput
        control={form.control}
        name="name"
        label="Project name"
        placeholder="e.g. Backend platform"
        autoFocus
      />
      <FormInput control={form.control} name="prefix" label="Prefix" placeholder="e.g. BE" maxLength={5} />
      <Button type="submit" className="w-full sm:w-auto" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting ? "Creating…" : "Continue"}
      </Button>
    </form>
  );
};
