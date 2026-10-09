import { useState } from "react";

import type { Conversation } from "@nexul/client-core/chat";

import { ChannelSettingsDialog } from "@/components/chat/ChannelSettingsDialog";
import { RenameChannelForm } from "@/components/chat/RenameChannelForm";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useDeleteChannel } from "@/hooks/ChatHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { channelHasSettingsCard, channelMention, SaveChannelFormSchema, type SaveChannelFormData } from "@/models/Chat";

// A channel row's Settings, Rename, and Delete, each undefined when the viewer has nothing there; #general is never
// deleted or made private. Settings opens settingsDialog, which the row renders.
export const useChannelRowActions = (conversation: Conversation) => {
  const can = useAreaAccess();
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteChannel = useDeleteChannel();
  const [settingsOpen, setSettingsOpen] = useState(false);
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

  // Bots read alone opens it too, on a channel with nothing else to set.
  const hasSettings = channelHasSettingsCard(conversation, !!can?.("editChannels")) || !!can?.("bots");

  return {
    onSettings: hasSettings ? () => setSettingsOpen(true) : undefined,
    onRename: can?.("editChannels") ? () => void rename() : undefined,
    onDelete: can?.("deleteChannels") && !conversation.general ? () => void remove() : undefined,
    settingsDialog: <ChannelSettingsDialog conversationId={settingsOpen ? conversation.id : null} onClose={() => setSettingsOpen(false)} />,
  };
};
