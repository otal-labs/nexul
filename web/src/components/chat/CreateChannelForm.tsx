import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateChannel } from "@/hooks/ChatHooks";
import { useCreateVoiceChannel } from "@/hooks/VoiceHooks";
import type { Conversation, SaveChannelFormData } from "@/models/Chat";

interface CreateChannelFormProps {
  workspaceId: string;
  /** Picks which create endpoint the submit hits. Default false (text channel). */
  voice?: boolean;
  onCreated?: (conversation: Conversation) => void;
}

export const CreateChannelForm = ({ workspaceId, voice = false, onCreated }: CreateChannelFormProps) => {
  const { control, onSubmit } = useFormDialogContext<SaveChannelFormData>();
  const createChannel = useCreateChannel(workspaceId);
  const createVoiceChannel = useCreateVoiceChannel(workspaceId);

  onSubmit(async (input) => {
    const conversation = await (voice ? createVoiceChannel : createChannel).mutateAsync(input.name);
    onCreated?.(conversation);
    return input;
  });

  return (
    <FormInput
      control={control}
      name="name"
      id="channel-name"
      label={voice ? "Voice channel name" : "Channel name"}
      placeholder={voice ? "e.g. huddle" : "e.g. incidents"}
      autoFocus
      autoComplete="off"
    />
  );
};
