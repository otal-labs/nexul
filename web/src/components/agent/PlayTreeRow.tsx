import { PLAY_STAGE_LABELS } from "@nexul/client-core/plays";

import { formatShortAgo } from "@/components/doc/docTime";
import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { PlayEnabledSwitch } from "@/components/play/PlayEnabledSwitch";
import { treeRowClass } from "@/components/agent/treeRow";
import { useOpenRow } from "@/hooks/useOpenRow";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { cn } from "@/lib/utils";
import type { WorkspacePlay } from "@/models/Play";

interface PlayTreeRowProps {
  play: WorkspacePlay;
  open: boolean;
  onHistory: () => void;
}

// A ticket play says where its button shows before its description; any other play is its description.
const lineOf = (play: WorkspacePlay) => {
  const description = play.description.trim();
  if (play.type !== "ticket" || !play.show_when_stage) return description;
  const where = `Button on tickets in ${PLAY_STAGE_LABELS[play.show_when_stage]}`;
  return description ? `${where} · ${description}` : where;
};

// One play: its name over one muted line, its runs in 7 days, its last run and its switch; the row unfolds its settings.
export const PlayTreeRow = ({ play, open, onHistory }: PlayTreeRowProps) => {
  const { toggle, setOpen } = useOpenRow();
  const menu = { ...usePlayMenu(play, onHistory), onEdit: () => setOpen(play.id) };

  return (
    <RowContextMenu menu={menu}>
      <div className={cn(treeRowClass, "py-2.5 pl-4", open && currentRowFillClass)}>
        {open && <CurrentRowEdge />}
        <button type="button" aria-expanded={open} onClick={() => toggle(play.id)} className="inset-focus min-w-0 rounded-md py-0.5 text-left">
          <span className={cn("block truncate text-sm font-medium", !play.enabled && "text-muted-foreground")}>{play.label}</span>
          <span className="block truncate text-xs text-muted-foreground">{lineOf(play)}</span>
        </button>
        <span className={cn("hidden text-right font-mono text-sm tabular-nums sm:block", play.runs_7d === 0 && "text-muted-foreground")}>{play.runs_7d}</span>
        <span className="hidden text-right font-mono text-xs text-muted-foreground tabular-nums sm:block">{play.last_run ? formatShortAgo(play.last_run.started_at) : "—"}</span>
        <PlayEnabledSwitch play={play} />
      </div>
    </RowContextMenu>
  );
};
