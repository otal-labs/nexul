import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { PlusIcon } from "lucide-react";
import { useForm } from "react-hook-form";

import { EnterList } from "@/components/EnterList";
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

// Adding a type isn't the card's main job, so its form waits behind the footer button.
export const BoardTicketTypesSection = ({ projectId, ticketTypes }: BoardTicketTypesSectionProps) => {
  const createTicketType = useCreateTicketType();
  const [adding, setAdding] = useState(false);

  const typeForm = useForm<SaveTicketTypeFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(SaveTicketTypeFormSchema),
  });

  return (
    <SettingsCard
      id="ticket-types"
      title="Ticket types"
      description="Each type has a color and a body template its new tickets start from."
      footer={
        !adding && (
          <Button variant="outline" size="sm" onClick={() => setAdding(true)}>
            <PlusIcon className="size-4" />
            New ticket type
          </Button>
        )
      }
    >
      {ticketTypes && ticketTypes.length === 0 && !adding && <EmptyRow>No ticket types yet. Add one to sort tickets by kind.</EmptyRow>}
      {ticketTypes && ticketTypes.length > 0 && (
        <EnterList className="divide-y divide-border rounded-md border border-border">
          {ticketTypes.map((type) => (
            <TicketTypeRow key={type.id} type={type} projectId={projectId} />
          ))}
        </EnterList>
      )}
      {adding && (
        <form
          className="settle-in mt-4 flex flex-wrap items-end gap-2"
          onSubmit={typeForm.handleSubmit(async (data) => {
            await createTicketType.mutateAsync({ project_id: projectId, name: data.name });
            typeForm.reset({ name: "" });
            setAdding(false);
          })}
        >
          <FormInput
            control={typeForm.control}
            name="name"
            id="new-ticket-type-name"
            label="New ticket type"
            placeholder="e.g. bug, feature, task"
            className="w-full sm:w-80"
            autoFocus
          />
          <Button type="submit" loading={typeForm.formState.isSubmitting}>
            Add type
          </Button>
          <Button type="button" variant="ghost" onClick={() => setAdding(false)}>
            Cancel
          </Button>
        </form>
      )}
    </SettingsCard>
  );
};
