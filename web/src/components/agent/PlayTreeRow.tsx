import { PLAY_STAGE_LABELS } from "@nexul/client-core/plays";

import { CurrentRowEdge, currentRowFillClass } from "@/components/CurrentRow";
import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { PlayEnabledSwitch } from "@/components/play/PlayEnabledSwitch";
import { lastRunDate } from "@/components/agent/lastRunDate";
import { RowChevron } from "@/components/agent/RowChevron";
import { stopRowClick, treeRowClass } from "@/components/agent/treeRow";
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

// One play: its name over one muted line, its runs in 7 days, its last run and its switch; the whole row opens its settings.
export const PlayTreeRow = ({ play, open, onHistory }: PlayTreeRowProps) => {
  const { toggle, setOpen } = useOpenRow();
  const menu = { ...usePlayMenu(play, onHistory), onEdit: () => setOpen(play.id) };

  return (
    <RowContextMenu menu={menu}>
      <div onClick={() => toggle(play.id)} className={cn(treeRowClass, "py-2.5 pl-3", open && currentRowFillClass)}>
        {open && <CurrentRowEdge />}
        <button type="button" aria-expanded={open} className="inset-focus flex min-w-0 items-start gap-1.5 rounded-md py-0.5 text-left">
          <RowChevron open={open} className="mt-0.5" />
          <span className="min-w-0">
            <span className={cn("block truncate text-sm font-medium", !play.enabled && "text-muted-foreground")}>{play.label}</span>
            <span className="line-clamp-2 text-xs text-muted-foreground">{lineOf(play)}</span>
          </span>
        </button>
        <span className={cn("hidden text-right font-mono text-sm tabular-nums sm:block", play.runs_7d === 0 ? "text-muted-foreground" : "text-foreground")}>{play.runs_7d}</span>
        <span className="hidden text-right font-mono text-xs text-muted-foreground tabular-nums lg:block">{play.last_run ? lastRunDate(play.last_run.started_at) : "—"}</span>
        <span onClick={stopRowClick} className="flex justify-end">
          <PlayEnabledSwitch play={play} />
        </span>
      </div>
    </RowContextMenu>
  );
};
