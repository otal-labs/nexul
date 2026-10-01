import { MoreHorizontalIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { PersonAvatar } from "@/components/PersonAvatar";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useRemoveChannelMember } from "@/hooks/ChannelHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { useOpenDirectMessage } from "@/hooks/useOpenDirectMessage";
import type { Conversation } from "@/models/Chat";
import { personLabel } from "@/models/Person";

interface ChannelMemberRowProps {
  channel: Conversation;
  userId: string;
  isYou: boolean;
  // Closes the channel's settings when a message opens elsewhere.
  onLeaveDialog: () => void;
}

export const ChannelMemberRow = ({ channel, userId, isYou, onLeaveDialog }: ChannelMemberRowProps) => {
  const person = usePerson(userId);
  const name = personLabel(person);
  const can = useAreaAccess();
  const removeMember = useRemoveChannelMember();
  const openDM = useOpenDirectMessage(userId);
  const canRemove = !isYou && (can?.("editChannels") ?? false);
  const message = !isYou && openDM;

  return (
    <li className="flex min-h-11 items-center gap-3">
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6 text-[10px]" />
      <span className="min-w-0 flex-1 truncate text-sm">{name}</span>
      {isYou && <span className="pr-2 font-mono text-[11px] text-muted-foreground">you</span>}
      {(message || canRemove) && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="size-8 text-muted-foreground" aria-label={`Actions for ${name}`}>
              <MoreHorizontalIcon className="size-4" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-44">
            {message && (
              <DropdownMenuItem
                onSelect={() => {
                  onLeaveDialog();
                  message();
                }}
              >
                Send a message
              </DropdownMenuItem>
            )}
            {message && canRemove && <DropdownMenuSeparator />}
            {canRemove && (
              <DropdownMenuItem variant="destructive" onSelect={() => removeMember.mutate({ channel, userId, name })}>
                Remove from channel
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </li>
  );
};
