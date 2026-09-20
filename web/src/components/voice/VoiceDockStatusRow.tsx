import { AlertTriangle, Loader2, PhoneOff } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import type { VoiceCallStatus } from "@/stores/voiceCallStore";

interface VoiceDockStatusRowProps {
  status: VoiceCallStatus;
  channelName: string;
  onLeave: () => void;
}

export const VoiceDockStatusRow = ({ status, channelName, onLeave }: VoiceDockStatusRowProps) => (
  <div className="flex items-center justify-between gap-2">
    <div className="min-w-0">
      {status === "connected" && <p className="text-xs font-medium text-success">Voice connected</p>}
      {(status === "connecting" || status === "idle") && (
        <p className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
          <Loader2 className="size-3 animate-spin motion-reduce:animate-none" aria-hidden />
          Connecting…
        </p>
      )}
      {status === "not_configured" && (
        <p className="text-xs font-medium text-muted-foreground">
          <Link to="/settings" className="underline underline-offset-2 hover:text-foreground">
            LiveKit setup needed
          </Link>
        </p>
      )}
      {status === "error" && (
        <p className="flex items-center gap-1.5 text-xs font-medium text-destructive">
          <AlertTriangle className="size-3" aria-hidden />
          Connection failed
        </p>
      )}
      <p className="truncate text-xs text-muted-foreground">{channelName}</p>
    </div>
    <Button variant="ghost" size="icon" className="shrink-0" aria-label="Leave voice" onClick={onLeave}>
      <PhoneOff className="size-4 text-destructive" aria-hidden />
    </Button>
  </div>
);
