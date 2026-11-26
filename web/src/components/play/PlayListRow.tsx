import { ZapIcon } from "lucide-react";

import { PLAY_STAGE_LABELS, playAttention } from "@nexul/client-core/plays";

import { formatShortAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { agentPath } from "@/models/Agent";
import { type WorkspacePlay } from "@/models/Play";

interface PlayListRowProps {
  play: WorkspacePlay;
  selected: boolean;
}

// The group already names the type, so a ticket play's line says where it shows and any other play's previews what it does.
const whereText = (play: WorkspacePlay) =>
  play.type === "ticket" && play.show_when_stage ? `On “${PLAY_STAGE_LABELS[play.show_when_stage]}”` : play.description.trim();

export const PlayListRow = ({ play, selected }: PlayListRowProps) => {
  const wsPath = useWorkspacePath();
  const menu = usePlayMenu(play);
  const attention = playAttention(play);
  const auto = (play.auto_plays_on ?? 0) > 0;
  const time = play.last_run && !attention?.state.startsWith("run") && (
    <span className="font-mono text-micro text-muted-foreground tabular-nums">{formatShortAgo(play.last_run.started_at)}</span>
  );

  // In a tight row the stage or preview cuts; the auto count and what needs a look never do.
  const where = whereText(play);
  const detail = (where || auto || !play.enabled || attention) && (
    <span className="flex w-full min-w-0 items-center gap-3">
      {where && <span className="min-w-0 truncate">{where}</span>}
      {auto && (
        <span className="inline-flex shrink-0 items-center gap-1">
          <ZapIcon aria-hidden className="size-3" />
          {play.auto_plays_on} <span>auto on</span>
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

  return <ListPaneRow to={wsPath(agentPath("plays", play.id))} title={play.label} muted={!play.enabled} detail={detail || undefined} dense={!detail} selected={selected} menu={menu} meta={time} metaOnTitleLine />;
};
