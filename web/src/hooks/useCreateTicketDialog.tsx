import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveTicketFormSchema, type SaveTicketFormData } from "@/models/Ticket";
import { loadTicketForm } from "@/utils/loadTicketForm";

interface CreateTicketOptions {
  // Links the new ticket to this doc.
  docId?: string;
  // Files the new ticket under this category.
  categoryId?: string;
}

// Opens the New ticket dialog in projectId and resolves with the new ticket's id; undefined for a viewer who may not
// write tickets there, so callers hide the action.
export const useCreateTicketDialog = (
  projectId: string,
): ((options?: CreateTicketOptions) => Promise<string | undefined>) | undefined => {
  const canCreate = useAreaAccess(projectId)?.("editTickets") ?? false;
  const { open } = useFormDialog();
  if (!canCreate) return undefined;
  return async ({ docId = "", categoryId = "" }: CreateTicketOptions = {}) => {
    const { CreateTicketForm, CreateTicketFooter, emptyTicketForm } = await loadTicketForm();
    const result = await open<SaveTicketFormData>({
      title: "New ticket",
      schema: SaveTicketFormSchema,
      okLabel: "Create ticket",
      header: <ProjectDialogHeader title="New ticket" />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm defaultProjectId={projectId} docId={docId} defaultCategoryId={categoryId} />,
      formOptions: { defaultValues: emptyTicketForm() },
    });
    return (result.data as { id?: string } | null)?.id;
  };
};
