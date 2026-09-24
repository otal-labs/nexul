import { formatUpdatedAgo } from "@/components/doc/docTime";
import { SetupRefusalLink } from "@/components/pairing/SetupRefusalLink";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { useChatAuthorLookup } from "@/hooks/ChatHooks";
import { useLiveTrailActivity, useLiveTrailState } from "@/hooks/TrailHooks";
import { SETUP_REQUIRED_REASON } from "@/models/Pairing";
import { trailSummary, type Trail } from "@/models/Trail";

interface TrailRowProps {
  workspaceId: string;
  trail: Trail;
  onOpen: (trailId: string) => void;
}

// One hairline row per run: state, play, a one-line summary, who pressed it, and when.
export const TrailRow = ({ workspaceId, trail, onOpen }: TrailRowProps) => {
  const state = useLiveTrailState(trail);
  const activity = useLiveTrailActivity(trail);
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const lastStep = trail.activity[trail.activity.length - 1];

  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(trail.id)}
        className="flex w-full items-center gap-2 px-2 py-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <TrailStateIcon state={state} />
        <span className="min-w-0 flex-1 truncate text-xs">
          <span className="font-medium">{trail.play_label}</span>
          <span className="text-muted-foreground"> · {trailSummary(state, trail.last_error, activity ?? lastStep)}</span>
        </span>
        <span className="shrink-0 font-mono text-[11px] text-muted-foreground">
          {resolveLogin(trail.starter_id)} · {formatUpdatedAgo(trail.started_at)}
        </span>
      </button>
      {trail.failure_reason === SETUP_REQUIRED_REASON && (
        <div className="px-2 pb-1.5">
          <SetupRefusalLink computerId={trail.computer_id} />
        </div>
      )}
    </li>
  );
};
