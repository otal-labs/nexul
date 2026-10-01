import { Controller } from "react-hook-form";

import { ChannelPeoplePicker } from "@/components/chat/ChannelPeoplePicker";
import { PrivateChannelRow } from "@/components/chat/PrivateChannelRow";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { useCreateChannel } from "@/hooks/ChatHooks";
import type { Conversation, CreateChannelFormData } from "@/models/Chat";

interface CreateChannelFormProps {
  workspaceId: string;
  /** Picks which create endpoint the submit hits. Default false (text channel). */
  voice?: boolean;
  // A Restricted member creates only private channels.
  privateOnly?: boolean;
  onCreated?: (conversation: Conversation) => void;
}

export const CreateChannelForm = ({ workspaceId, voice = false, privateOnly = false, onCreated }: CreateChannelFormProps) => {
  const { control, onSubmit, watch } = useFormDialogContext<CreateChannelFormData>();
  const createChannel = useCreateChannel(workspaceId);
  const isPrivate = watch("private");

  onSubmit(async (input) => {
    const conversation = await createChannel.mutateAsync({ ...input, voice });
    onCreated?.(conversation);
    return input;
  });

  return (
    <div className="min-w-0 space-y-2">
      <FormInput
        control={control}
        name="name"
        id="channel-name"
        label={voice ? "Voice channel name" : "Channel name"}
        placeholder={voice ? "e.g. huddle" : "e.g. incidents"}
        autoFocus
        autoComplete="off"
      />
      <Controller
        control={control}
        name="private"
        render={({ field }) => <PrivateChannelRow checked={field.value} onCheckedChange={field.onChange} fixed={privateOnly} />}
      />
      {isPrivate && (
        <Controller
          control={control}
          name="member_ids"
          render={({ field }) => <ChannelPeoplePicker value={field.value} onChange={field.onChange} />}
        />
      )}
    </div>
  );
};
