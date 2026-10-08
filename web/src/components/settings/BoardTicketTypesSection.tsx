import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { EmptyRow } from "@/components/EmptyRow";
import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
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
    <SettingsCard
      id="ticket-types"
      title="Ticket types"
      description="What a ticket on this board can be. Each type carries a color and a body template new tickets start from."
    >
      {ticketTypes && ticketTypes.length === 0 && <EmptyRow>No ticket types yet</EmptyRow>}
      {ticketTypes && ticketTypes.length > 0 && (
        <ul className="divide-y divide-border">
          {ticketTypes.map((type) => (
            <TicketTypeRow key={type.id} type={type} projectId={projectId} />
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
        <Button type="submit" loading={typeForm.formState.isSubmitting}>
          Add type
        </Button>
      </form>
    </SettingsCard>
  );
};
