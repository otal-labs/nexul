import { CloneTemplateForm } from "@/components/templates/CloneTemplateForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { CloneTemplateFormSchema, type CloneTemplateFormData, type CloneSource } from "@/models/Template";

export const useCloneTemplateDialog = () => {
  const { open } = useFormDialog();
  return (source: CloneSource) =>
    open<CloneTemplateFormData>({
      title: `Clone ${source.name} to…`,
      description: "Copies this text over the template where you pick.",
      schema: CloneTemplateFormSchema,
      okLabel: "Clone",
      form: <CloneTemplateForm source={source} />,
      formOptions: { defaultValues: { scope: "", target: "" } },
    });
};
