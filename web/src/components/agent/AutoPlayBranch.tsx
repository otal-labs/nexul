import { AutoPlayTreeRow } from "@/components/agent/AutoPlayTreeRow";
import { AutoPlayEditor } from "@/components/autoplay/AutoPlayEditor";
import { useOpenRow } from "@/hooks/useOpenRow";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

interface AutoPlayBranchProps {
  play: Play;
  autoPlay: AutoPlay;
  open: boolean;
}

export const AutoPlayBranch = ({ play, autoPlay, open }: AutoPlayBranchProps) => {
  const { setOpen } = useOpenRow();
  return (
    <div>
      <AutoPlayTreeRow play={play} autoPlay={autoPlay} open={open} />
      {open && <AutoPlayEditor key={autoPlay.id} play={play} autoPlay={autoPlay} onClose={() => setOpen("")} />}
    </div>
  );
};
