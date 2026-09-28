import { CreateChannelForm } from "@/components/chat/CreateChannelForm";
import { CreateDMForm } from "@/components/chat/CreateDMForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import {
  SaveChannelFormSchema,
  SaveDMFormSchema,
  type Conversation,
  type SaveChannelFormData,
  type SaveDMFormData,
} from "@/models/Chat";

// The create dialogs for a channel, a voice channel and a direct message, shared by the chat list and the sidebar.
export const useNewConversationDialogs = (workspaceId: string, onCreated: (conversation: Conversation) => void) => {
  const { open } = useFormDialog();

  const openNewChannel = (voice: boolean) =>
    open<SaveChannelFormData>({
      title: voice ? "New voice channel" : "New channel",
      schema: SaveChannelFormSchema,
      okLabel: voice ? "Create voice channel" : "Create channel",
      form: <CreateChannelForm workspaceId={workspaceId} voice={voice} onCreated={onCreated} />,
      formOptions: { defaultValues: { name: "" } },
    });

  const openNewDM = () =>
    open<SaveDMFormData>({
      title: "New direct message",
      schema: SaveDMFormSchema,
      okLabel: "Start conversation",
      form: <CreateDMForm workspaceId={workspaceId} onCreated={onCreated} />,
      formOptions: { defaultValues: { participant_ids: [] } },
    });

  return { openNewChannel, openNewDM };
};
