import { Plus } from "lucide-react";

import { CreateChannelForm } from "@/components/chat/CreateChannelForm";
import { CreateDMForm } from "@/components/chat/CreateDMForm";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveChannelFormSchema, SaveDMFormSchema, type SaveChannelFormData, type SaveDMFormData } from "@/models/Chat";

interface NewConversationMenuProps {
  workspaceId: string;
  onCreated: (conversationId: string) => void;
}

export const NewConversationMenu = ({ workspaceId, onCreated }: NewConversationMenuProps) => {
  const { open } = useFormDialog();

  const openNewChannel = (voice: boolean) =>
    open<SaveChannelFormData>({
      title: voice ? "New voice channel" : "New channel",
      schema: SaveChannelFormSchema,
      okLabel: voice ? "Create voice channel" : "Create channel",
      form: <CreateChannelForm workspaceId={workspaceId} voice={voice} onCreated={(c) => onCreated(c.id)} />,
      formOptions: { defaultValues: { name: "" } },
    });

  const openNewDM = () =>
    open<SaveDMFormData>({
      title: "New direct message",
      schema: SaveDMFormSchema,
      okLabel: "Start conversation",
      form: <CreateDMForm workspaceId={workspaceId} onCreated={(c) => onCreated(c.id)} />,
      formOptions: { defaultValues: { participant_ids: [] } },
    });

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8" aria-label="New conversation">
          <Plus className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => void openNewChannel(false)}>New channel</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => void openNewChannel(true)}>New voice channel</DropdownMenuItem>
        <DropdownMenuItem onSelect={() => void openNewDM()}>New direct message</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
