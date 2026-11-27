import { useState } from "react";

import { PlacementRow } from "@/components/agent/PlacementRow";
import { Button } from "@/components/ui/button";
import type { PlacementAnswer } from "@/models/Pairing";
import type { WorkspacePlay } from "@/models/Play";

// Past this many, the rest wait behind Show more, so a long list of plays never buries the settings above it.
const SHOWN = 5;

export interface PlacedPlay {
  play: WorkspacePlay;
  answer: PlacementAnswer;
}

interface PlacementListProps {
  placed: PlacedPlay[];
  nameOf: (computerId: string) => string;
}

export const PlacementList = ({ placed, nameOf }: PlacementListProps) => {
  const [all, setAll] = useState(false);
  const shown = all ? placed : placed.slice(0, SHOWN);
  const hidden = placed.length - shown.length;

  return (
    <div className="space-y-2">
      <ul className="divide-y divide-border rounded-md border border-border">
        {shown.map(({ play, answer }) => (
          <PlacementRow key={play.id} play={play} answer={answer} nameOf={nameOf} />
        ))}
      </ul>
      {hidden > 0 && (
        <Button type="button" variant="ghost" size="sm" onClick={() => setAll(true)}>
          Show {hidden} more
        </Button>
      )}
    </div>
  );
};
