import { useState } from "react";

import { PlayRunDialog } from "@/components/play/PlayRunDialog";
import { Button } from "@/components/ui/button";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { PlayQueueItem } from "@/models/PlayQueue";

interface RunItButtonProps {
  item: PlayQueueItem;
}

// Runs the play by hand for the viewer through the usual run dialog, so the press keeps its permission and stage checks.
export const RunItButton = ({ item }: RunItButtonProps) => {
  const canRun = useHasPermission("plays:run");
  const { data: plays } = useFetchWorkspacePlays(item.workspace_id);
  const play = plays?.find((p) => p.id === item.play_id);
  const [open, setOpen] = useState(false);

  return (
    canRun &&
    play && (
      <span className="shrink-0">
        <Button variant="outline" size="sm" className="h-7 px-2 text-xs" aria-label={`Run ${item.play_label}`} onClick={() => setOpen(true)}>
          Run it
        </Button>
        <PlayRunDialog
          play={play}
          projectId={item.project_id}
          targetType={item.target_type}
          targetId={item.target_id}
          open={open}
          onClose={() => setOpen(false)}
        />
      </span>
    )
  );
};
