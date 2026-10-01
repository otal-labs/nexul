import { Controller } from "react-hook-form";

import { ChannelPeoplePicker } from "@/components/chat/ChannelPeoplePicker";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useSetChannelPrivate } from "@/hooks/ChannelHooks";
import type { ChannelPeopleFormData, Conversation } from "@/models/Chat";

interface WhoStaysFormProps {
  channel: Conversation;
}

// Turning a channel private keeps the person switching and whoever is checked; everyone else loses it at once.
export const WhoStaysForm = ({ channel }: WhoStaysFormProps) => {
  const { control, onSubmit } = useFormDialogContext<ChannelPeopleFormData>();
  const setPrivate = useSetChannelPrivate();

  onSubmit(async (input) => {
    await setPrivate.mutateAsync({ id: channel.id, isPrivate: true, memberIds: input.user_ids });
    return input;
  });

  return (
    <Controller
      control={control}
      name="user_ids"
      render={({ field }) => <ChannelPeoplePicker value={field.value} onChange={field.onChange} />}
    />
  );
};
