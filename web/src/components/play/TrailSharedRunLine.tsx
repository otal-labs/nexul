import { personLabel } from "@nexul/client-core/person";

import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { Computer } from "@/models/Pairing";
import type { Trail } from "@/models/Trail";

interface TrailSharedRunLineProps {
  trail: Trail;
  computer: Computer;
}

// Who ran it on whose computer, when they are two people: the run used a share of someone else's computer.
export const TrailSharedRunLine = ({ trail, computer }: TrailSharedRunLineProps) => {
  const { data: me } = useFetchMe();
  const lookup = usePersonLookup(trail.workspace_id);
  const myID = me?.user?.id;
  const ownComputerOthersRun = !computer.shared && !!myID && trail.starter_id !== myID;
  const owner = computer.shared?.owner_name ?? (ownComputerOthersRun && myID ? personLabel(lookup(myID)) : "");
  if (!owner) return null;
  return (
    <span className="text-xs text-muted-foreground">
      Run by {personLabel(lookup(trail.starter_id))} using {owner}'s {computer.name}
    </span>
  );
};
