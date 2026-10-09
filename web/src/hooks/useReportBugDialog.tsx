import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useFormDialog } from "@/hooks/useFormDialog";
import { emptyTicketForm, ReportBugFormSchema, type SaveTicketFormData } from "@/models/Ticket";
import { loadTicketForm } from "@/utils/loadTicketForm";

interface ReportBugOptions {
  originId?: string;
  projectId?: string;
}

// Report a bug from a ticket pre-fills the found-in link; from the board the origin is picked or marked unknown.
export const useReportBugDialog = () => {
  const { open } = useFormDialog();
  return async (options: ReportBugOptions = {}) => {
    const { CreateTicketForm, CreateTicketFooter } = await loadTicketForm();
    return open<SaveTicketFormData>({
      title: "Report a bug",
      schema: ReportBugFormSchema,
      okLabel: "Report bug",
      header: <ProjectDialogHeader title="Report a bug" />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm bug allowOriginUnknown={!options.originId} defaultProjectId={options.projectId ?? ""} />,
      formOptions: { defaultValues: { ...emptyTicketForm(), origin_id: options.originId ?? "", origin_unknown: false } },
    });
  };
};
