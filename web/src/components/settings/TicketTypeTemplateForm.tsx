import { Textarea } from "@/components/ui/textarea";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useSetTicketTypeTemplate } from "@/hooks/TicketTypeHooks";
import type { SaveTicketTypeTemplateFormData } from "@/models/TicketType";

interface TicketTypeTemplateFormProps {
  typeId: string;
}

export const TicketTypeTemplateForm = ({ typeId }: TicketTypeTemplateFormProps) => {
  const { register, onSubmit } = useFormDialogContext<SaveTicketTypeTemplateFormData>();
  const setTemplate = useSetTicketTypeTemplate();

  onSubmit(async (input) => {
    const type = await setTemplate.mutateAsync({ id: typeId, body_template: input.body_template });
    return { body_template: type.body_template };
  });

  return (
    <div className="space-y-1.5">
      <label htmlFor="ticket-type-template" className="text-sm font-medium">
        Template (markdown)
      </label>
      <Textarea
        id="ticket-type-template"
        {...register("body_template")}
        autoFocus
        rows={12}
        placeholder={"## Why\n\n## Acceptance criteria"}
        className="font-mono text-xs"
      />
    </div>
  );
};
