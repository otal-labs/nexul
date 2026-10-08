import { CreateTicketFooter } from "@/components/ticket/CreateTicketFooter";
import { CreateTicketForm, emptyTicketForm } from "@/components/ticket/CreateTicketForm";
import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveTicketFormSchema, type SaveTicketFormData } from "@/models/Ticket";

// Opens the New ticket dialog in projectId and resolves with the new ticket's id; undefined for a viewer who may not
// write tickets there, so callers hide the action.
export const useCreateTicketDialog = (projectId: string): (() => Promise<string | undefined>) | undefined => {
  const canCreate = useAreaAccess(projectId)?.("editTickets") ?? false;
  const { open } = useFormDialog();
  if (!canCreate) return undefined;
  return async () => {
    const result = await open<SaveTicketFormData>({
      title: "New ticket",
      schema: SaveTicketFormSchema,
      okLabel: "Create",
      header: <ProjectDialogHeader title="New ticket" />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm defaultProjectId={projectId} />,
      formOptions: { defaultValues: emptyTicketForm() },
    });
    return (result.data as { id?: string } | null)?.id;
  };
};
