import { Controller } from "react-hook-form";

import { ChannelPeoplePicker } from "@/components/chat/ChannelPeoplePicker";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useAddChannelMembers } from "@/hooks/ChannelHooks";
import type { ChannelPeopleFormData, Conversation } from "@/models/Chat";

interface AddChannelPeopleFormProps {
  channel: Conversation;
}

export const AddChannelPeopleForm = ({ channel }: AddChannelPeopleFormProps) => {
  const { control, onSubmit, formState } = useFormDialogContext<ChannelPeopleFormData>();
  const addMembers = useAddChannelMembers();
  const error = formState.errors.user_ids;

  onSubmit(async (input) => {
    await addMembers.mutateAsync({ id: channel.id, userIds: input.user_ids });
    return input;
  });

  return (
    <div className="space-y-2">
      <Controller
        control={control}
        name="user_ids"
        render={({ field }) => (
          <ChannelPeoplePicker
            value={field.value}
            onChange={field.onChange}
            excludeIds={channel.participant_ids ?? []}
            keepsViewer={false}
          />
        )}
      />
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error.message}
        </p>
      )}
    </div>
  );
};
