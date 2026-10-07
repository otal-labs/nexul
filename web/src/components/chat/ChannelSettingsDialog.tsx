import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { BotsSection } from "@/components/botwebhook/BotsSection";
import { ChannelSettingsCard } from "@/components/chat/ChannelSettingsCard";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { channelHasSettingsCard, channelMention } from "@/models/Chat";
import { cn } from "@/lib/utils";

interface ChannelSettingsDialogProps {
  conversationId: string | null;
  onClose: () => void;
}

// Reads the channel from the sidebar's own query, so a live change shows here and a channel the viewer loses closes it.
export const ChannelSettingsDialog = ({ conversationId, onClose }: ChannelSettingsDialogProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(workspaceId);
  const can = useAreaAccess();
  const channel = conversations?.find((c) => c.id === conversationId);
  const hasCard = !!channel && channelHasSettingsCard(channel, !!can?.("editChannels"));
  const hasBots = !!can?.("bots");

  return (
    <Dialog open={!!channel} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="flex max-h-[min(90dvh,52rem)] flex-col sm:max-w-md">
        {channel && (
          <DialogHeader className="min-w-0 shrink-0 pr-8">
            <DialogTitle className="break-words">{channelMention(channel)}</DialogTitle>
            <DialogDescription>{hasCard ? "Who sees it and who is in it." : "The bots that post here."}</DialogDescription>
          </DialogHeader>
        )}
        <div className="-mx-6 min-h-0 flex-1 space-y-4 overflow-y-auto px-6 py-0.5">
          {channel && hasCard && <ChannelSettingsCard channel={channel} />}
          {channel && hasBots && <BotsSection conversation={channel} className={cn(hasCard && "border-t border-border pt-4")} />}
        </div>
        <DialogFooter className="shrink-0">
          <DialogClose asChild>
            <Button size="sm">Done</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
