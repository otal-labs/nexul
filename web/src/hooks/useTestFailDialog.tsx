import { TestFailForm } from "@/components/ticket/TestFailForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { emptyTestFailForm, TestFailFormSchema, type TestFailFormData } from "@/models/TicketTest";

// A failed test is a bug found before done: it goes to the ticket's thread, never a new bug ticket (ADR 0064).
export const useTestFailDialog = () => {
  const { open } = useFormDialog();
  return (ticketId: string) =>
    open<TestFailFormData>({
      title: "What went wrong?",
      description: "This goes to the ticket's thread and moves the ticket back to progress.",
      schema: TestFailFormSchema,
      okLabel: "Send back",
      form: <TestFailForm ticketId={ticketId} />,
      formOptions: { defaultValues: emptyTestFailForm() },
    });
};
