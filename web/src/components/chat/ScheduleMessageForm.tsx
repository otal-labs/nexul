import { REPEAT_LABELS } from "@nexul/client-core/calendar";
import type { ScheduledMessage } from "@nexul/client-core/chat";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { useScheduleMessage, useUpdateScheduledMessage } from "@/hooks/ScheduledMessageHooks";
import { scheduleChange, scheduleInput, type ScheduleFormData } from "@/models/ScheduledMessage";

const REPEAT_OPTIONS = Object.entries(REPEAT_LABELS).map(([value, label]) => ({ value, label }));

interface ScheduleMessageFormProps {
  conversationId: string;
  // editing is the scheduled message being changed; without it the form schedules a new one.
  editing?: ScheduledMessage;
}

// The message, the day and time it posts, in the browser's zone, and how often it posts again.
export const ScheduleMessageForm = ({ conversationId, editing }: ScheduleMessageFormProps) => {
  const { control, onSubmit } = useFormDialogContext<ScheduleFormData>();
  const schedule = useScheduleMessage(conversationId);
  const update = useUpdateScheduledMessage();

  onSubmit(async (form) => {
    if (editing) await update.mutateAsync({ id: editing.id, change: scheduleChange(editing, form) });
    if (!editing) await schedule.mutateAsync(scheduleInput(form));
    return form;
  });

  return (
    <div className="min-w-0 space-y-4">
      <FormTextarea control={control} name="body" label="Message" rows={4} className="border-input bg-transparent" autoFocus />
      <div className="grid grid-cols-[1.4fr_1fr] items-start gap-3">
        <FormInput control={control} name="date" type="date" label="Day" />
        <FormInput control={control} name="time" type="time" label="Time" />
      </div>
      <FormSelect control={control} name="repeats" label="Repeats" options={REPEAT_OPTIONS} />
    </div>
  );
};
