import { LogOutIcon, UserPlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ChannelMemberRow } from "@/components/chat/ChannelMemberRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useChannelSettingsActions } from "@/hooks/useChannelSettingsActions";
import type { Conversation } from "@/models/Chat";

interface ChannelMembersSectionProps {
  channel: Conversation;
  onLeaveDialog: () => void;
}

// A private channel's members: anyone in it adds people, and anyone but the last member leaves.
export const ChannelMembersSection = ({ channel, onLeaveDialog }: ChannelMembersSectionProps) => {
  const { data: me } = useFetchMe();
  const { addPeople, leave } = useChannelSettingsActions(channel);
  const memberIds = channel.participant_ids ?? [];
  const isMember = !!me && memberIds.includes(me.user.id);

  return (
    <section aria-label="Members">
      <div className="flex min-h-11 items-center justify-between gap-2 border-b border-border">
        <p className="font-mono text-[11px] tracking-wide text-muted-foreground uppercase">
          {memberIds.length} {memberIds.length === 1 ? "member" : "members"}
        </p>
        <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={addPeople}>
          <UserPlusIcon className="size-4" aria-hidden />
          Add people
        </Button>
      </div>
      <ul className="divide-y divide-border border-b border-border">
        {memberIds.map((userId) => (
          <ChannelMemberRow key={userId} channel={channel} userId={userId} isYou={userId === me?.user.id} onLeaveDialog={onLeaveDialog} />
        ))}
      </ul>
      {isMember && memberIds.length > 1 && (
        <Button variant="ghost" size="sm" className="mt-3 -ml-2 text-destructive hover:bg-destructive/10 hover:text-destructive" onClick={leave}>
          <LogOutIcon className="size-4" aria-hidden />
          Leave channel
        </Button>
      )}
    </section>
  );
};
