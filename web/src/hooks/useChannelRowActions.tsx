import { useState, type ComponentProps, type ComponentType } from "react";

import type { Conversation } from "@nexul/client-core/chat";

import type { ChannelSettingsDialog } from "@/components/chat/ChannelSettingsDialog";
import { RenameChannelForm } from "@/components/chat/RenameChannelForm";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useRowGates } from "@/hooks/useRowGates";
import { useDeleteChannel } from "@/hooks/ChatHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useFormDialog } from "@/hooks/useFormDialog";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { channelHasSettingsCard, channelMention, SaveChannelFormSchema, type SaveChannelFormData } from "@/models/Chat";
import { channelEntity } from "@/models/PermissionPanel";
import { loadChannelSettingsDialog } from "@/utils/loadAccessDialogs";

// A channel row's Settings, Rename, Permissions… and Delete, each undefined when the viewer has nothing there; #general
// is never deleted or made private. Settings opens settingsDialog, which the row renders.
export const useChannelRowActions = (conversation: Conversation) => {
  const can = useAreaAccess();
  const { offered, gates, deleteGate } = useRowGates({
    onRename: { permission: "channels:write", verb: "rename channels" },
    onDelete: { permission: "channels:delete" },
  });
  const { open: openForm } = useFormDialog();
  const { open: confirm } = useConfirmationDialog();
  const deleteChannel = useDeleteChannel();
  const openPermissions = usePermissionsDialog();
  const [settingsOpen, setSettingsOpen] = useState(false);
  // Mounted from the first open on, so every channel row doesn't carry the dialog and the close still animates.
  const [SettingsDialog, setSettingsDialog] = useState<ComponentType<ComponentProps<typeof ChannelSettingsDialog>>>();
  const label = channelMention(conversation);
  const entity = channelEntity(conversation, label);

  const rename = () =>
    openForm<SaveChannelFormData>({
      title: `Rename ${label}`,
      schema: SaveChannelFormSchema,
      okLabel: "Rename",
      form: <RenameChannelForm conversation={conversation} />,
      formOptions: { defaultValues: { name: conversation.name ?? "" } },
    });

  const openSettings = async () => {
    const dialog = await loadChannelSettingsDialog();
    setSettingsDialog(() => dialog);
    setSettingsOpen(true);
  };

  const remove = async () => {
    const ok = await confirm({ title: `Delete ${label}?`, message: "Its messages are deleted for good.", confirmLabel: "Delete" });
    if (ok) deleteChannel.mutate(conversation);
  };

  // Bots read alone opens it too, on a channel with nothing else to set.
  const hasSettings = channelHasSettingsCard(conversation, offered("onRename")) || !!can?.("bots");

  return {
    onSettings: hasSettings ? () => void openSettings() : undefined,
    gates,
    deleteGate,
    onRename: offered("onRename") ? () => void rename() : undefined,
    permissions: { entity, onOpen: () => void openPermissions(entity) },
    onDelete: offered("onDelete") && !conversation.general ? () => void remove() : undefined,
    settingsDialog: SettingsDialog && <SettingsDialog conversationId={settingsOpen ? conversation.id : null} onClose={() => setSettingsOpen(false)} />,
  };
};
