import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { VoiceOccupant } from "@/models/Voice";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceOccupantListProps {
  occupants: VoiceOccupant[];
}

export const VoiceOccupantList = ({ occupants }: VoiceOccupantListProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  if (occupants.length === 0) return null;

  return (
    <div className="flex flex-col gap-0.5" aria-label={`In call: ${occupants.map((o) => o.name).join(", ")}`}>
      {occupants.map((occupant) => (
        // pl-[3.25rem] = navLinkClass px-2.5 + w-8 icon column + gap-2.5, so the avatar starts where the channel name does.
        <div
          key={occupant.identity}
          className="flex min-w-0 items-center gap-2.5 rounded-md py-1 pr-2.5 pl-[3.25rem] text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground"
          title={occupant.name}
        >
          <PersonAvatar login={lookup(occupant.identity).login} src={lookup(occupant.identity).avatar_url} />
          <span className="min-w-0 truncate">{occupant.name}</span>
        </div>
      ))}
    </div>
  );
};
