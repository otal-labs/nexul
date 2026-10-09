import type { Conversation } from "@nexul/client-core/chat";

import { CreateChannelForm } from "@/components/chat/CreateChannelForm";
import { CreateDMForm } from "@/components/chat/CreateDMForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { CreateChannelFormSchema, SaveDMFormSchema, type CreateChannelFormData, type SaveDMFormData } from "@/models/Chat";

// The create dialogs for a channel, a voice channel and a direct message, shared by the chat list and the sidebar.
export const useNewConversationDialogs = (workspaceId: string, onCreated: (conversation: Conversation) => void) => {
  const { open } = useFormDialog();
  const { data: role } = useFetchMyRole(workspaceId);
  const privateOnly = role?.restricted === true;

  const openNewChannel = (voice: boolean) =>
    open<CreateChannelFormData>({
      title: voice ? "New voice channel" : "New channel",
      schema: CreateChannelFormSchema,
      okLabel: voice ? "Create voice channel" : "Create channel",
      form: <CreateChannelForm workspaceId={workspaceId} voice={voice} privateOnly={privateOnly} onCreated={onCreated} />,
      formOptions: { defaultValues: { name: "", private: privateOnly, member_ids: [] } },
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
