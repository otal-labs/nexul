import { ShieldIcon } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { Button } from "@/components/ui/button";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { channelMention } from "@/models/Chat";
import { channelEntity } from "@/models/PermissionPanel";
import { cn } from "@/lib/utils";

interface ChannelAccessRowProps {
  channel: Conversation;
  // Closes the settings first, so the panel doesn't open over a second dialog.
  onOpen: () => void;
  className?: string;
}

// Who sees and who posts in the channel, set in its permissions panel by role or person.
export const ChannelAccessRow = ({ channel, onOpen, className }: ChannelAccessRowProps) => {
  const openPermissions = usePermissionsDialog();
  const entity = channelEntity(channel, channelMention(channel));
  return (
    <div className={cn("flex items-center gap-3 py-3", className)}>
      <ShieldIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-sm font-medium">Who sees and posts here</p>
        <p className="text-sm text-muted-foreground">Rules for a role or a person on this channel.</p>
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          onOpen();
          void openPermissions(entity);
        }}
      >
        Permissions…
      </Button>
    </div>
  );
};
