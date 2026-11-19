import { ZapIcon } from "lucide-react";

import { formatShortAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_STAGE_LABELS, playAttention, type WorkspacePlay } from "@/models/Play";

interface PlayListRowProps {
  play: WorkspacePlay;
  selected: boolean;
}

// The group already names the type, so a row says only where a ticket play shows.
const whereText = (play: WorkspacePlay) => play.type === "ticket" && play.show_when_stage && `On “${PLAY_STAGE_LABELS[play.show_when_stage]}”`;

export const PlayListRow = ({ play, selected }: PlayListRowProps) => {
  const wsPath = useWorkspacePath();
  const menu = usePlayMenu(play);
  const attention = playAttention(play);
  const auto = (play.auto_plays_on ?? 0) > 0;
  const time = play.last_run && !attention?.state.startsWith("run") && (
    <span className="font-mono text-micro text-muted-foreground tabular-nums">{formatShortAgo(play.last_run.started_at)}</span>
  );

  // In a tight row the auto count drops its words first, then the stage cuts; the count and what needs a look never do.
  const where = whereText(play);
  const detail = (where || auto || !play.enabled || attention) && (
    <span className="@container flex w-full min-w-0 items-center gap-3">
      {where && <span className="min-w-0 truncate">{where}</span>}
      {auto && (
        <span className="inline-flex shrink-0 items-center gap-1">
          <ZapIcon aria-hidden className="size-3" />
          {play.auto_plays_on} <span className="@max-[20rem]:sr-only">auto on</span>
        </span>
      )}
      {!play.enabled && <span className="shrink-0">Off</span>}
      {attention && (
        <span className="inline-flex shrink-0 items-center gap-1 text-foreground">
          <TrailStateIcon state={attention.state} className="size-3" />
          <span>{attention.text}</span>
        </span>
      )}
    </span>
  );

  return <ListPaneRow to={wsPath(`/plays/${play.id}`)} title={play.label} muted={!play.enabled} detail={detail || undefined} dense={!detail} selected={selected} menu={menu} meta={time} metaOnTitleLine />;
};
