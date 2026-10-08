import { formatUpdatedAgo } from "@/components/doc/docTime";
import { SetupRefusalLink } from "@/components/pairing/SetupRefusalLink";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useLiveTrailActivity, useLiveTrailState } from "@/hooks/TrailHooks";
import { SETUP_REQUIRED_REASON } from "@/models/Pairing";
import { personLabel } from "@/models/Person";
import { trailSummary, type Trail } from "@/models/Trail";
import { cn } from "@/lib/utils";

interface TrailRowProps {
  workspaceId: string;
  trail: Trail;
  onOpen: (trailId: string) => void;
  /** "stacked" puts who and when under the summary, for a column too narrow to hold both on one line. */
  layout?: "inline" | "stacked";
}

// One hairline row per run: state, play, a one-line summary, who pressed it, and when.
export const TrailRow = ({ workspaceId, trail, onOpen, layout = "inline" }: TrailRowProps) => {
  const state = useLiveTrailState(trail);
  const activity = useLiveTrailActivity(trail);
  const lookup = usePersonLookup(workspaceId);
  const lastStep = trail.activity[trail.activity.length - 1];
  const stacked = layout === "stacked";

  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(trail.id)}
        className={cn(
          "flex w-full gap-2 px-2 py-1.5 text-left transition-colors duration-150 ease-standard hover:bg-accent/40",
          stacked ? "items-start" : "items-center",
        )}
      >
        <TrailStateIcon state={state} className={cn(stacked && "mt-0.5")} />
        <span className={cn("flex min-w-0 flex-1 gap-2", stacked ? "flex-col gap-0.5" : "items-center")}>
          <span className={cn("min-w-0 truncate text-xs", !stacked && "flex-1")}>
            <span className="font-medium">{trail.play_label}</span>
            <span className="text-muted-foreground"> · {trailSummary(state, trail.last_error, activity ?? lastStep)}</span>
          </span>
          <span className={cn("font-mono text-xs text-muted-foreground", stacked ? "truncate" : "shrink-0")}>
            {personLabel(lookup(trail.starter_id))} · {formatUpdatedAgo(trail.started_at)}
          </span>
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
