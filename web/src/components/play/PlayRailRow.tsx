import { formatUpdatedAgo } from "@/components/doc/docTime";
import { PlayButton } from "@/components/play/PlayButton";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useLiveTrailState } from "@/hooks/TrailHooks";
import type { ListedPlay } from "@/models/Play";
import type { Ticket } from "@/models/Ticket";
import type { Trail } from "@/models/Trail";

interface PlayRailRowProps {
  play: ListedPlay;
  ticket: Ticket;
  /** The play's newest run on this ticket, if it has one. */
  lastRun: Trail | undefined;
}

// One line per play: its name, then how its last run on this ticket went and when.
export const PlayRailRow = ({ play, ticket, lastRun }: PlayRailRowProps) => (
  <div className="flex min-w-0 items-center gap-2 pr-2">
    <PlayButton
      play={play}
      projectId={ticket.project_id}
      targetType="ticket"
      targetId={ticket.id}
      variant="ghost"
      className="min-w-0 flex-1"
      mainClassName="h-7 min-w-0 flex-1 justify-start gap-2 px-2! text-left text-xs font-normal"
    />
    {lastRun && <LastRun trail={lastRun} />}
  </div>
);

const LastRun = ({ trail }: { trail: Trail }) => {
  const state = useLiveTrailState(trail);
  return (
    <span className="flex shrink-0 items-center gap-1.5 font-mono text-xs text-muted-foreground tabular-nums">
      <TrailStateIcon state={state} className="size-3" />
      {formatUpdatedAgo(trail.started_at)}
    </span>
  );
};
