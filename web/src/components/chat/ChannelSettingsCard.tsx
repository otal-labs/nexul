import { ChannelMembersSection } from "@/components/chat/ChannelMembersSection";
import { PrivateChannelRow } from "@/components/chat/PrivateChannelRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useChannelSettingsActions } from "@/hooks/useChannelSettingsActions";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Conversation } from "@/models/Chat";
import { cn } from "@/lib/utils";

interface ChannelSettingsCardProps {
  channel: Conversation;
  onClose: () => void;
}

// The private switch over a private channel's members; a Restricted member's channel stays private.
export const ChannelSettingsCard = ({ channel, onClose }: ChannelSettingsCardProps) => {
  const can = useAreaAccess();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: role } = useFetchMyRole(workspaceId);
  const { setPrivate } = useChannelSettingsActions(channel);
  const canSwitch = (can?.("editChannels") ?? false) && !channel.general;
  const isPrivate = channel.private ?? false;

  return (
    <div className={cn("rounded-lg border border-border bg-card px-4", isPrivate && "pb-4")}>
      {canSwitch && (
        <PrivateChannelRow
          checked={isPrivate}
          fixed={isPrivate && role?.restricted === true}
          onCheckedChange={setPrivate}
          className={cn(isPrivate && "border-b border-border")}
        />
      )}
      {isPrivate && <ChannelMembersSection channel={channel} onLeaveDialog={onClose} />}
    </div>
  );
};
