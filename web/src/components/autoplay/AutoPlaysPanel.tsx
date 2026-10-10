import { useCallback, useState } from "react";

import { AutoPlayComposer } from "@/components/autoplay/AutoPlayComposer";
import { AutoPlaysList } from "@/components/autoplay/AutoPlaysList";
import type { AutoPlay } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

// The list, or one auto play's composer in its place (null opens a new one).
export const AutoPlaysPanel = ({ play }: { play: Play }) => {
  const [open, setOpen] = useState<{ autoPlay: AutoPlay | null } | null>(null);
  const openAutoPlay = useCallback((autoPlay: AutoPlay | null) => setOpen({ autoPlay }), []);

  return (
    <div className="min-w-0">
      {open && <AutoPlayComposer play={play} autoPlay={open.autoPlay} onClose={() => setOpen(null)} />}
      {!open && <AutoPlaysList play={play} onOpen={openAutoPlay} />}
    </div>
  );
};
