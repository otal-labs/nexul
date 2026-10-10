import { useRef } from "react";

import { AutoPlaysPanel } from "@/components/autoplay/AutoPlaysPanel";
import { PlayForm } from "@/components/play/PlayForm";
import { useSwapEntrance } from "@/hooks/useSwapEntrance";
import { usePlayDialogStore } from "@/stores/playDialogStore";
import type { Play } from "@/models/Play";

interface PlayDialogBodyProps {
  workspaceId: string;
  play: Play;
}

// The edit dialog's open tab; the Play form keeps its values in the dialog's form while Auto plays is showing.
export const PlayDialogBody = ({ workspaceId, play }: PlayDialogBodyProps) => {
  const tab = usePlayDialogStore((s) => s.tab);
  const panel = useRef<HTMLDivElement>(null);
  useSwapEntrance(tab, () => (tab === "play" ? 0 : 1), () => panel.current, () => "x");

  return (
    <div ref={panel} role="tabpanel" aria-label={tab === "play" ? "Play" : "Auto plays"} className="pt-1">
      {tab === "play" && <PlayForm workspaceId={workspaceId} editing={play} />}
      {tab === "auto" && <AutoPlaysPanel play={play} />}
    </div>
  );
};
