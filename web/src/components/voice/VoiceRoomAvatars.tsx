import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import type { VoiceOccupant } from "@/models/Voice";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceRoomAvatarsProps {
  occupants: VoiceOccupant[];
  /** Identities speaking right now; each ring in the success hue, as in the call. */
  speaking?: ReadonlySet<string>;
}

const SHOWN = 5;

// Who is in the room before you join: a stack of up to five, then a +N count.
export const VoiceRoomAvatars = ({ occupants, speaking }: VoiceRoomAvatarsProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const rest = occupants.length - SHOWN;
  return (
    <div className="flex shrink-0 items-center -space-x-1">
      {occupants.slice(0, SHOWN).map((occupant) => (
        <PersonAvatar
          key={occupant.identity}
          login={lookup(occupant.identity).login}
          name={occupant.name}
          src={lookup(occupant.identity).avatar_url}
          label={occupant.name}
          className={cn(
            "size-7 text-[11px] ring-2 ring-surface-2 transition-shadow duration-150 ease-standard",
            speaking?.has(occupant.identity) && "ring-success",
          )}
        />
      ))}
      {rest > 0 && (
        <span className="avatar-more size-7 text-[11px] ring-2 ring-surface-2">
          +{rest}
        </span>
      )}
    </div>
  );
};
