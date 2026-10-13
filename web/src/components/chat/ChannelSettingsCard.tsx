import type { Conversation } from "@nexul/client-core/chat";

import { ChannelMembersSection } from "@/components/chat/ChannelMembersSection";
import { PrivateChannelRow } from "@/components/chat/PrivateChannelRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useChannelSettingsActions } from "@/hooks/useChannelSettingsActions";
import { cn } from "@/lib/utils";

interface ChannelSettingsCardProps {
  channel: Conversation;
}

// The private switch over a private channel's members.
export const ChannelSettingsCard = ({ channel }: ChannelSettingsCardProps) => {
  const can = useAreaAccess();
  const { setPrivate } = useChannelSettingsActions(channel);
  const canSwitch = (can?.("editChannels") ?? false) && !channel.general;
  const isPrivate = channel.private ?? false;

  return (
    <div className={cn("min-w-0 rounded-lg border border-border bg-card px-4", isPrivate && "pb-4")}>
      {canSwitch && (
        <PrivateChannelRow
          checked={isPrivate}
          onCheckedChange={setPrivate}
          className={cn(isPrivate && "border-b border-border")}
        />
      )}
      {isPrivate && <ChannelMembersSection channel={channel} />}
    </div>
  );
};
