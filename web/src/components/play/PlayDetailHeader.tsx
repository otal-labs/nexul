import { useState } from "react";

import { PageHeader } from "@/components/PageHeader";
import { ActionButton } from "@/components/action/ActionButton";
import { RowActions } from "@/components/listpane/RowActions";
import { offersMenu } from "@/components/listpane/rowMenu";
import { PlayHistoryDialog } from "@/components/play/PlayHistoryDialog";
import { PlayEnabledSwitch } from "@/components/play/PlayEnabledSwitch";
import { PlayHealthLine } from "@/components/play/PlayHealthLine";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import { useEntityActions } from "@/hooks/useEntityActions";
import { usePlayDialogs } from "@/hooks/usePlayDialogs";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { PLAY_TYPE_LABELS, type WorkspacePlay } from "@/models/Play";

interface PlayDetailHeaderProps {
  play: WorkspacePlay;
}

export const PlayDetailHeader = ({ play }: PlayDetailHeaderProps) => {
  const wsPath = useWorkspacePath();
  const crumbs = useEmbeddedCrumbs([{ label: "Plays", to: wsPath("/plays") }, { label: `${PLAY_TYPE_LABELS[play.type]} play` }]);
  const answer = useEntityActions("play", play.id);
  const { edit } = usePlayDialogs();
  const [historyOpen, setHistoryOpen] = useState(false);
  const menu = { ...usePlayMenu(play, () => setHistoryOpen(true)), onEdit: undefined };

  return (
    <PageHeader
      crumbs={crumbs}
      title={play.label}
      meta={
        <div className="flex min-w-0 flex-col gap-2">
          {play.description && <span className="max-w-prose text-pretty">{play.description}</span>}
          <PlayHealthLine health={play} />
        </div>
      }
      actions={
        <>
          <PlayEnabledSwitch play={play} />
          <span aria-hidden className="mx-1 h-5 w-px bg-border" />
          <ActionButton permission="plays:write" answer={answer("plays:write")} verb="edit plays" variant="outline" size="sm" onClick={() => void edit(play)}>
            Edit
          </ActionButton>
          {offersMenu(menu, !!menu.permissions) && <RowActions itemLabel={play.label} {...menu} />}
          {historyOpen && <PlayHistoryDialog playId={play.id} name={play.label} onClose={() => setHistoryOpen(false)} />}
        </>
      }
    />
  );
};
