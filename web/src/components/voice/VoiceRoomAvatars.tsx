import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { VoiceOccupant } from "@/models/Voice";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceRoomAvatarsProps {
  occupants: VoiceOccupant[];
}

const SHOWN = 5;

// Who is in the room before you join: a stack of up to five, then a +N count.
export const VoiceRoomAvatars = ({ occupants }: VoiceRoomAvatarsProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const rest = occupants.length - SHOWN;
  return (
    <div className="flex shrink-0 items-center -space-x-2">
      {occupants.slice(0, SHOWN).map((occupant) => (
        <PersonAvatar
          key={occupant.identity}
          login={lookup(occupant.identity).login}
          src={lookup(occupant.identity).avatar_url}
          label={occupant.name}
          className="size-7 text-[11px] ring-2 ring-surface-2"
        />
      ))}
      {rest > 0 && (
        <span className="flex size-7 items-center justify-center rounded-full bg-muted font-mono text-[11px] text-muted-foreground tabular-nums ring-2 ring-surface-2">
          +{rest}
        </span>
      )}
    </div>
  );
};
