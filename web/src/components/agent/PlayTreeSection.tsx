import { useState } from "react";

import { AutoPlayBranch } from "@/components/agent/AutoPlayBranch";
import { NewAutoPlayBranch } from "@/components/agent/NewAutoPlayBranch";
import { PlayEditor } from "@/components/agent/PlayEditor";
import { PlayTreeRow } from "@/components/agent/PlayTreeRow";
import { PlayQueueNowLine } from "@/components/autoplay/PlayQueueNowLine";
import { PlayHistoryDialog } from "@/components/play/PlayHistoryDialog";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchAutoPlays } from "@/hooks/PlayHooks";
import { useOpenRow } from "@/hooks/useOpenRow";
import type { WorkspacePlay } from "@/models/Play";

// One play and its branch: the play's row, then its settings when it is open, then a row per auto play and the way to add one; an interview play has no branch.
export const PlayTreeSection = ({ play }: { play: WorkspacePlay }) => {
  const { open, setOpen } = useOpenRow();
  const can = useAreaAccess();
  const [historyOpen, setHistoryOpen] = useState(false);
  const branch = play.type !== "interview" && !!can?.("autoPlays");
  const { data: autoPlays } = useFetchAutoPlays(play.workspace_id, branch ? play.id : "");

  return (
    <section aria-label={play.label} className="group/play relative">
      {open === play.id && <span aria-hidden className="pointer-events-none absolute inset-y-0 left-0 z-10 w-0.5 bg-brand" />}
      <PlayTreeRow play={play} open={open === play.id} onHistory={() => setHistoryOpen(true)} />
      {open === play.id && <PlayEditor key={play.id} play={play} onClose={() => setOpen("")} onHistory={() => setHistoryOpen(true)} />}
      {branch && autoPlays?.map((autoPlay) => <AutoPlayBranch key={autoPlay.id} play={play} autoPlay={autoPlay} open={open === autoPlay.id} />)}
      {branch && autoPlays && <NewAutoPlayBranch play={play} shown={open === play.id || autoPlays.length === 0} />}
      {branch && (
        <div className="bg-surface-2 py-2 pl-12 empty:hidden">
          <PlayQueueNowLine play={play} />
        </div>
      )}
      {historyOpen && <PlayHistoryDialog playId={play.id} name={play.label} onClose={() => setHistoryOpen(false)} />}
    </section>
  );
};
