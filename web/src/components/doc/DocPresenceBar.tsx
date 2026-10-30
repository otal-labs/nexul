import { PresenceSection } from "@/components/doc/collab/PresenceSection";
import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { cn } from "@/lib/utils";

interface DocPresenceBarProps {
  participants: CollabParticipant[];
  connected: boolean | undefined;
  updatedAt: string;
  size?: "sm" | "md";
}

export const DocPresenceBar = ({ participants, connected, updatedAt, size }: DocPresenceBarProps) => (
  <div className="flex items-center gap-3">
    <PresenceSection participants={participants} max={3} {...(size ? { size } : {})} />
    {connected != null && (
      <span className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
        <span
          className={cn(
            "size-1.5 rounded-full",
            connected ? "animate-[status-pulse_2.4s_ease-standard_infinite] bg-success motion-reduce:animate-none" : "bg-muted-foreground/40",
          )}
          aria-hidden="true"
        />
        {connected ? "live" : "offline"}
      </span>
    )}
    <span className="font-mono text-xs text-muted-foreground tabular-nums">updated {formatUpdatedAgo(updatedAt)}</span>
  </div>
);
