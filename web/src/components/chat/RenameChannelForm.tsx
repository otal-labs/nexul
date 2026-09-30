import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useRenameChannel } from "@/hooks/ChatHooks";
import type { Conversation, SaveChannelFormData } from "@/models/Chat";

interface RenameChannelFormProps {
  conversation: Conversation;
}

export const RenameChannelForm = ({ conversation }: RenameChannelFormProps) => {
  const { control, onSubmit } = useFormDialogContext<SaveChannelFormData>();
  const renameChannel = useRenameChannel();

  onSubmit(async (input) => {
    await renameChannel.mutateAsync({ id: conversation.id, name: input.name });
    return input;
  });

  return (
    <FormInput
      control={control}
      name="name"
      id="channel-name"
      label={conversation.kind === "voice_channel" ? "Voice channel name" : "Channel name"}
      autoFocus
      autoComplete="off"
    />
  );
};
