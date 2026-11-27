import { newAutoPlayKey, NewAutoPlayRow } from "@/components/agent/NewAutoPlayRow";
import { AutoPlayEditor } from "@/components/autoplay/AutoPlayEditor";
import { useOpenRow } from "@/hooks/useOpenRow";
import type { Play } from "@/models/Play";

// "+ Start it by itself when…", and under it a new auto play's editor while it is the open row.
export const NewAutoPlayBranch = ({ play }: { play: Play }) => {
  const { open, setOpen } = useOpenRow();
  const unfolded = open === newAutoPlayKey(play.id);
  return (
    <div>
      <NewAutoPlayRow playId={play.id} open={unfolded} />
      {unfolded && <AutoPlayEditor play={play} autoPlay={null} onClose={() => setOpen("")} />}
    </div>
  );
};
