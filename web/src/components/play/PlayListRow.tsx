import { ZapIcon } from "lucide-react";

import { formatShortAgo } from "@/components/doc/docTime";
import { ListPaneRow } from "@/components/listpane/ListPaneRow";
import { TrailStateIcon } from "@/components/play/TrailStateIcon";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_STAGE_LABELS, PLAY_TYPE_LABELS, playAttention, type WorkspacePlay } from "@/models/Play";

interface PlayListRowProps {
  play: WorkspacePlay;
  selected: boolean;
}

const whereText = (play: WorkspacePlay) => (play.type === "ticket" && play.show_when_stage ? `On ${PLAY_STAGE_LABELS[play.show_when_stage]}` : PLAY_TYPE_LABELS[play.type]);

const Sep = () => <span aria-hidden>·</span>;

export const PlayListRow = ({ play, selected }: PlayListRowProps) => {
  const wsPath = useWorkspacePath();
  const menu = usePlayMenu(play);
  const attention = playAttention(play);
  const auto = (play.auto_plays_on ?? 0) > 0;
  const time = play.last_run && !attention?.state.startsWith("run") && (
    <span className="font-mono text-[11px] text-muted-foreground tabular-nums">{formatShortAgo(play.last_run.started_at)}</span>
  );

  const detail = (
    <>
      <span className="min-w-0 truncate">{whereText(play)}</span>
      {auto && (
        <>
          <Sep />
          <span className="inline-flex shrink-0 items-center gap-1">
            <ZapIcon aria-hidden className="size-3" />
            {play.auto_plays_on} auto on
          </span>
        </>
      )}
      {!play.enabled && (
        <>
          <Sep />
          <span className="shrink-0">Off</span>
        </>
      )}
      {attention && (
        <>
          <Sep />
          <span className="inline-flex shrink-0 items-center gap-1 text-foreground">
            <TrailStateIcon state={attention.state} className="size-3" />
            <span>{attention.text}</span>
          </span>
        </>
      )}
    </>
  );

  return <ListPaneRow to={wsPath(`/plays/${play.id}`)} title={play.label} muted={!play.enabled} detail={detail} selected={selected} menu={menu} meta={time} metaOnTitleLine />;
};
