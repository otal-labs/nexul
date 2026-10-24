import { REMIND_OPTIONS, type Meeting } from "@nexul/client-core/calendar";

import { MeetingInviteesField } from "@/components/calendar/MeetingInviteesField";
import { MeetingRepeatFields } from "@/components/calendar/MeetingRepeatFields";
import { MeetingWhenFields } from "@/components/calendar/MeetingWhenFields";
import { MeetingWhereFields } from "@/components/calendar/MeetingWhereFields";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { FormTextarea } from "@/components/ticket/FormTextarea";
import { useCreateMeeting, useUpdateMeeting } from "@/hooks/CalendarHooks";
import { meetingBody, type MeetingFormData, type MeetingScope } from "@/models/Meeting";
import { useWorkspaceStore } from "@/stores/workspaceStore";

const REMIND_CHOICES = REMIND_OPTIONS.map((o) => ({ value: String(o.minutes), label: o.label }));

// A change to a meeting that already exists: which occurrence, and whether it is alone or with the ones after it.
export interface MeetingEdit {
  meeting: Meeting;
  slot: string;
  // "all" is a change with no occurrence named, which covers the whole meeting.
  scope: MeetingScope | "all";
}

interface MeetingFormProps {
  editing?: MeetingEdit;
  onSaved: (meeting: Meeting) => void;
}

export const MeetingForm = ({ editing, onSaved }: MeetingFormProps) => {
  const { control, onSubmit } = useFormDialogContext<MeetingFormData>();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const create = useCreateMeeting(workspaceId);
  const update = useUpdateMeeting();
  // One occurrence on its own cannot repeat, so its repeat fields are not offered.
  const alone = editing?.scope === "this";

  onSubmit(async (input) => {
    if (!editing) {
      onSaved(await create.mutateAsync(meetingBody(input, "create")));
      return input;
    }
    const named = editing.scope === "all" ? {} : { slot: editing.slot, scope: editing.scope };
    onSaved(await update.mutateAsync({ id: editing.meeting.id, change: { ...meetingBody(input, alone ? "one" : "change"), ...named } }));
    return input;
  });

  return (
    <div className="min-w-0 space-y-4">
      <FormInput control={control} name="title" label="Title" autoComplete="off" autoFocus />
      <MeetingWhenFields />
      {!alone && <MeetingRepeatFields />}
      <MeetingWhereFields />
      <MeetingInviteesField />
      <FormSelect control={control} name="remind_minutes" label="Reminder" options={REMIND_CHOICES} />
      <FormTextarea control={control} name="notes" label="Notes" rows={3} className="border-input bg-transparent" />
    </div>
  );
};
