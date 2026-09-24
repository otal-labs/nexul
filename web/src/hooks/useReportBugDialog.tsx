import { CreateTicketFooter } from "@/components/ticket/CreateTicketFooter";
import { CreateTicketForm, emptyTicketForm } from "@/components/ticket/CreateTicketForm";
import { CreateTicketHeader } from "@/components/ticket/CreateTicketHeader";
import { useFormDialog } from "@/hooks/useFormDialog";
import { ReportBugFormSchema, type SaveTicketFormData } from "@/models/Ticket";

interface ReportBugOptions {
  originId?: string;
  projectId?: string;
}

// Report a bug from a ticket pre-fills the found-in link; from the board the origin is picked or marked unknown.
export const useReportBugDialog = () => {
  const { open } = useFormDialog();
  return (options: ReportBugOptions = {}) =>
    open<SaveTicketFormData>({
      title: "Report a bug",
      schema: ReportBugFormSchema,
      okLabel: "Report bug",
      header: <CreateTicketHeader title="Report a bug" />,
      footerStart: <CreateTicketFooter />,
      form: <CreateTicketForm bug allowOriginUnknown={!options.originId} defaultProjectId={options.projectId ?? ""} />,
      formOptions: { defaultValues: { ...emptyTicketForm(), origin_id: options.originId ?? "", origin_unknown: false } },
    });
};
