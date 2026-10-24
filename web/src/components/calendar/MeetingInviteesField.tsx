import { Controller } from "react-hook-form";

import { ChannelPeoplePicker } from "@/components/chat/ChannelPeoplePicker";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useFetchMe } from "@/hooks/AuthHooks";
import type { MeetingFormData } from "@/models/Meeting";

// Any member of the workspace; the organiser is in the meeting already and is left out of the list.
export const MeetingInviteesField = () => {
  const { control } = useFormDialogContext<MeetingFormData>();
  const { data: me } = useFetchMe();
  return (
    <div className="space-y-1">
      <p className="text-sm font-medium">Invitees</p>
      <Controller
        control={control}
        name="invitee_ids"
        render={({ field }) => (
          <ChannelPeoplePicker value={field.value} onChange={field.onChange} excludeIds={me ? [me.user.id] : []} keepsViewer={false} />
        )}
      />
    </div>
  );
};
