import { PresenceSection } from "@/components/doc/collab/PresenceSection";
import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface DocPresenceBarProps {
  participants: CollabParticipant[];
  connected: boolean | undefined;
  updatedAt: string;
}

export const DocPresenceBar = ({ participants, connected, updatedAt }: DocPresenceBarProps) => (
  <div className="flex items-center gap-3">
    <PresenceSection participants={participants} max={3} />
    {connected != null && (
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            tabIndex={0}
            role="img"
            aria-label={connected ? "Live" : "Offline"}
            className="flex size-4 items-center justify-center rounded-sm"
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                connected
                  ? "animate-[status-pulse_2.4s_ease-standard_infinite] bg-success motion-reduce:animate-none"
                  : "bg-muted-foreground/40",
              )}
              aria-hidden="true"
              data-testid="live-dot"
            />
          </span>
        </TooltipTrigger>
        <TooltipContent>{connected ? "Live" : "Offline"}</TooltipContent>
      </Tooltip>
    )}
    <span className="font-mono text-xs text-muted-foreground tabular-nums">updated {formatUpdatedAgo(updatedAt)}</span>
  </div>
);
