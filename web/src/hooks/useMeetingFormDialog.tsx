import type { Meeting } from "@nexul/client-core/calendar";

import { MeetingForm, type MeetingEdit } from "@/components/calendar/MeetingForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useMeetingParams } from "@/hooks/useMeetingParams";
import { MeetingFormSchema, meetingFormDefaults, newMeetingDefaults, type MeetingFormData } from "@/models/Meeting";

const dialogClassName = "sm:max-w-xl";

// The panel follows a change to the meeting it landed on: a copy of one occurrence, or the series that carries on.
export const useMoveToMeeting = () => {
  const { show } = useMeetingParams();
  return (meeting: Meeting, named: boolean) => show(meeting.id, named && meeting.repeats !== "none" ? meeting.starts_at : undefined);
};

// The New meeting and Edit dialogs; both open the panel on the meeting they saved.
export const useMeetingFormDialog = () => {
  const { open } = useFormDialog();
  const moveTo = useMoveToMeeting();

  const create = () =>
    open<MeetingFormData>({
      title: "New meeting",
      schema: MeetingFormSchema,
      form: <MeetingForm onSaved={(saved) => moveTo(saved, false)} />,
      formOptions: { defaultValues: newMeetingDefaults() },
      okLabel: "Create",
      dialogClassName,
    });

  const edit = (editing: MeetingEdit) =>
    open<MeetingFormData>({
      title: "Edit meeting",
      schema: MeetingFormSchema,
      form: <MeetingForm editing={editing} onSaved={(saved) => moveTo(saved, editing.scope !== "all")} />,
      formOptions: { defaultValues: meetingFormDefaults(editing.meeting, editing.slot) },
      dialogClassName,
    });

  return { create, edit };
};
