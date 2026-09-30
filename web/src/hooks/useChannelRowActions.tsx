import { RenameChannelForm } from "@/components/chat/RenameChannelForm";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useDeleteChannel } from "@/hooks/ChatHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { channelMention, SaveChannelFormSchema, type Conversation, type SaveChannelFormData } from "@/models/Chat";

// A channel row's Rename and Delete, each undefined when the viewer's role lacks it; #general is never deleted.
export const useChannelRowActions = (conversation: Conversation) => {
  const can = useAreaAccess();
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteChannel = useDeleteChannel();
  const label = channelMention(conversation);

  const rename = () =>
    openForm<SaveChannelFormData>({
      title: `Rename ${label}`,
      schema: SaveChannelFormSchema,
      okLabel: "Rename",
      form: <RenameChannelForm conversation={conversation} />,
      formOptions: { defaultValues: { name: conversation.name ?? "" } },
    });

  const remove = async () => {
    const ok = await confirm({ title: `Delete ${label}?`, message: "Its messages are deleted for good.", confirmLabel: "Delete" });
    if (ok) deleteChannel.mutate(conversation);
  };

  return {
    onRename: can?.("editChannels") ? () => void rename() : undefined,
    onDelete: can?.("deleteChannels") && !conversation.general ? () => void remove() : undefined,
  };
};
