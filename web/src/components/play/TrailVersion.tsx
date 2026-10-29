import { useState } from "react";

import { PlayHistoryDialog } from "@/components/play/PlayHistoryDialog";
import { useFetchHistory } from "@/hooks/HistoryHooks";
import { playHistoryUrl } from "@/hooks/PlayHistoryHooks";
import type { Trail } from "@/models/Trail";

interface TrailVersionProps {
  trail: Trail;
  version: number;
}

// The instructions version a run used. It opens the play's history at that version while the version is still kept, and
// says so when a prune has dropped it; until the list answers, it is plain text.
export const TrailVersion = ({ trail, version }: TrailVersionProps) => {
  const [open, setOpen] = useState(false);
  const { data: history } = useFetchHistory(playHistoryUrl(trail.play_id));
  const kept = history?.versions.some((v) => v.version === version);

  return (
    <>
      {kept === false && <span className="text-muted-foreground">Version {version} is no longer kept</span>}
      {kept === undefined && <span>Ran with version {version}</span>}
      {kept && (
        <button type="button" onClick={() => setOpen(true)} className="inset-focus rounded-sm underline decoration-muted-foreground underline-offset-4 transition-colors duration-150 ease-standard hover:decoration-foreground">
          Ran with version {version}
        </button>
      )}
      {open && <PlayHistoryDialog playId={trail.play_id} name={trail.play_label} version={version} onClose={() => setOpen(false)} />}
    </>
  );
};
