import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useNewConversationDialogs } from "@/hooks/useNewConversationDialogs";

interface NewConversationMenuProps {
  workspaceId: string;
  onCreated: (conversationId: string) => void;
}

export const NewConversationMenu = ({ workspaceId, onCreated }: NewConversationMenuProps) => {
  const { openNewChannel, openNewDM } = useNewConversationDialogs(workspaceId, (c) => onCreated(c.id));
  const canCreate = useAreaAccess()?.("newConversation") ?? false;

  if (!canCreate) return null;
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
