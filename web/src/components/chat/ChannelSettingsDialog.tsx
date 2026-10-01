import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ChannelSettingsCard } from "@/components/chat/ChannelSettingsCard";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { channelMention } from "@/models/Chat";

interface ChannelSettingsDialogProps {
  conversationId: string | null;
  onClose: () => void;
}

// Reads the channel from the sidebar's own query, so a live change shows here and a channel the viewer loses closes it.
export const ChannelSettingsDialog = ({ conversationId, onClose }: ChannelSettingsDialogProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const channel = conversations?.find((c) => c.id === conversationId);

  return (
    <Dialog open={!!channel} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {channel && (
          <DialogHeader className="min-w-0 pr-8">
            <DialogTitle className="break-words">{channelMention(channel)}</DialogTitle>
            <DialogDescription>Who sees it and who is in it.</DialogDescription>
          </DialogHeader>
        )}
        {channel && <ChannelSettingsCard channel={channel} />}
        <DialogFooter>
          <DialogClose asChild>
            <Button size="sm">Done</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
