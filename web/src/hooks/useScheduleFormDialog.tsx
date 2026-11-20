import type { ScheduledMessage } from "@nexul/client-core/chat";

import { ScheduleMessageForm } from "@/components/chat/ScheduleMessageForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { ScheduleFormSchema, newScheduleDefaults, scheduleFormDefaults, type ScheduleFormData } from "@/models/ScheduledMessage";

// The Schedule message and Edit scheduled message dialogs; each resolves true once it saved.
export const useScheduleFormDialog = (conversationId: string) => {
  const { open } = useFormDialog();

  const create = async (body: string) =>
    (
      await open<ScheduleFormData>({
        title: "Schedule message",
        description: "It posts as you at the time you pick. Only you see it until then.",
        schema: ScheduleFormSchema,
        form: <ScheduleMessageForm conversationId={conversationId} />,
        formOptions: { defaultValues: newScheduleDefaults(body) },
        okLabel: "Schedule",
      })
    ).success;

  const edit = async (scheduled: ScheduledMessage) =>
    (
      await open<ScheduleFormData>({
        title: "Edit scheduled message",
        schema: ScheduleFormSchema,
        form: <ScheduleMessageForm conversationId={conversationId} editing={scheduled} />,
        formOptions: { defaultValues: scheduleFormDefaults(scheduled) },
        okLabel: "Save",
      })
    ).success;

  return { create, edit };
};
