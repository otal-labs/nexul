import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { TicketTypeRow } from "@/components/settings/TicketTypeRow";
import { Button } from "@/components/ui/button";
import { useCreateTicketType } from "@/hooks/TicketTypeHooks";
import { SaveTicketTypeFormSchema, type SaveTicketTypeFormData } from "@/models/TicketType";
import type { TicketType } from "@/models/TicketType";

interface BoardTicketTypesSectionProps {
  projectId: string;
  ticketTypes: TicketType[] | undefined;
}

export const BoardTicketTypesSection = ({ projectId, ticketTypes }: BoardTicketTypesSectionProps) => {
  const createTicketType = useCreateTicketType();

  const typeForm = useForm<SaveTicketTypeFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(SaveTicketTypeFormSchema),
  });

  return (
    <div className="border-t pt-6">
      <h3 className="text-sm font-semibold">Ticket types</h3>
      {ticketTypes && ticketTypes.length === 0 && (
        <p className="mt-3 text-sm text-muted-foreground">No ticket types yet</p>
      )}
      {ticketTypes && ticketTypes.length > 0 && (
        <ul className="mt-3 divide-y divide-border">
          {ticketTypes.map((type) => (
            <TicketTypeRow key={type.id} type={type} />
          ))}
        </ul>
      )}

      <form
        className="mt-4 flex flex-wrap items-end gap-2"
        onSubmit={typeForm.handleSubmit(async (data) => {
          await createTicketType.mutateAsync({ project_id: projectId, name: data.name });
          typeForm.reset({ name: "" });
        })}
      >
        <FormInput
          control={typeForm.control}
          name="name"
          id="new-ticket-type-name"
          label="New ticket type"
          placeholder="New ticket type (bug, feature, task, ...)"
          className="w-full sm:w-80"
        />
        <Button type="submit" disabled={typeForm.formState.isSubmitting}>
          Add type
        </Button>
      </form>
    </div>
  );
};
