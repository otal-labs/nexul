import type { Conversation } from "@nexul/client-core/chat";

import { ChannelAccessRow } from "@/components/chat/ChannelAccessRow";
import { ChannelMembersSection } from "@/components/chat/ChannelMembersSection";
import { PrivateChannelRow } from "@/components/chat/PrivateChannelRow";
import { isOffered } from "@/components/listpane/rowMenu";
import { useActionState } from "@/hooks/useActionState";
import { useChannelSettingsActions } from "@/hooks/useChannelSettingsActions";
import { cn } from "@/lib/utils";

interface ChannelSettingsCardProps {
  channel: Conversation;
  // Closes the settings as the permissions panel opens.
  onOpenPermissions: () => void;
}

// Who sees and posts here, the private switch, then a private channel's members.
export const ChannelSettingsCard = ({ channel, onOpenPermissions }: ChannelSettingsCardProps) => {
  const channelWrite = useActionState({ permission: "channels:write" });
  const { setPrivate } = useChannelSettingsActions(channel);
  const canManage = isOffered(channelWrite);
  const canSwitch = canManage && !channel.general;
  const isPrivate = channel.private ?? false;

  return (
    <div className={cn("min-w-0 rounded-lg border border-border bg-card px-4", isPrivate && "pb-4")}>
      {canManage && <ChannelAccessRow channel={channel} onOpen={onOpenPermissions} className={cn((canSwitch || isPrivate) && "border-b border-border")} />}
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
