import { ConnectionQuality } from "livekit-client";
import { SignalLow } from "lucide-react";

import { isWeakConnection } from "@/lib/connectionQuality";
import { cn } from "@/lib/utils";

interface VoiceConnectionMarkProps {
  quality: ConnectionQuality;
  className?: string;
}

// Nothing while the connection is good; muted when it is poor, destructive once lost.
export const VoiceConnectionMark = ({ quality, className }: VoiceConnectionMarkProps) => {
  if (!isWeakConnection(quality)) return null;
  const lost = quality === ConnectionQuality.Lost;
  return (
    <SignalLow
      className={cn("size-3 shrink-0", lost ? "text-destructive" : "text-muted-foreground", className)}
      aria-label={lost ? "Connection lost" : "Weak connection"}
    />
  );
};
