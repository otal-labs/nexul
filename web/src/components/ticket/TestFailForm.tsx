import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { ScreenshotField } from "@/components/ticket/ScreenshotField";
import { useTestFail } from "@/hooks/TicketTestHooks";
import type { TestFailFormData } from "@/models/TicketTest";

interface TestFailFormProps {
  ticketId: string;
}

const textareaClass = "border-input bg-transparent text-base sm:text-sm";

// The bug template's sections; the result posts to the ticket's thread, where the fixing agent reads it.
export const TestFailForm = ({ ticketId }: TestFailFormProps) => {
  const { control, onSubmit } = useFormDialogContext<TestFailFormData>();
  const fail = useTestFail();

  onSubmit(async (report) => {
    await fail.mutateAsync({ id: ticketId, report });
    return report;
  });

  return (
    <div className="space-y-4">
      <FormTextarea control={control} name="steps" label="Steps to reproduce" rows={3} className={textareaClass} />
      <FormTextarea control={control} name="expected" label="Expected result" rows={2} className={textareaClass} />
      <FormTextarea control={control} name="actual" label="Actual result" rows={2} className={textareaClass} />
      <div className="space-y-2">
        <p className="text-sm font-medium">Screenshot</p>
        <ScreenshotField ticketId={ticketId} />
      </div>
    </div>
  );
};
