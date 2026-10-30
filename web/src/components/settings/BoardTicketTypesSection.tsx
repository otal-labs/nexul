import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { PlusIcon } from "lucide-react";
import { useForm } from "react-hook-form";

import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsRowListClass } from "@/components/settings/SettingsRow";
import { TicketTypeRow } from "@/components/settings/TicketTypeRow";
import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { useCreateTicketType, useFetchProjectTicketTypes } from "@/hooks/TicketTypeHooks";
import { SaveTicketTypeFormSchema, type SaveTicketTypeFormData } from "@/models/TicketType";

interface BoardTicketTypesSectionProps {
  projectId: string;
}

// Adding a type isn't the card's main job, so its form waits behind the footer button.
export const BoardTicketTypesSection = ({ projectId }: BoardTicketTypesSectionProps) => {
  const { data: ticketTypes, isPending, error } = useFetchProjectTicketTypes(projectId);
  const createTicketType = useCreateTicketType();
  const [adding, setAdding] = useState(false);

  const typeForm = useForm<SaveTicketTypeFormData>({
    defaultValues: { name: "" },
    resolver: zodResolver(SaveTicketTypeFormSchema),
  });

  return (
    <SettingsCard
      id="ticket-types"
      title="Colors and templates"
      description="Each type has a color and a body template its new tickets start from."
      footer={
        !adding && (
          <ActionButton permission="projects:write" projectId={projectId} verb="add a ticket type" variant="outline" size="sm" icon={PlusIcon} onClick={() => setAdding(true)}>
            New ticket type
          </ActionButton>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {ticketTypes && ticketTypes.length === 0 && !adding && <EmptyRow>No ticket types yet. Add one to sort tickets by kind.</EmptyRow>}
      {ticketTypes && ticketTypes.length > 0 && (
        <EnterList className={settingsRowListClass}>
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
