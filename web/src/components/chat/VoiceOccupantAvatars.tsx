import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import type { VoiceOccupant } from "@/models/Voice";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceOccupantListProps {
  occupants: VoiceOccupant[];
  /** Left indent under the channel name; row geometry differs per surface, so the caller supplies it. */
  className?: string;
}

export const VoiceOccupantList = ({ occupants, className }: VoiceOccupantListProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  if (occupants.length === 0) return null;

  return (
    <div className={cn("flex flex-col gap-0.5 pb-1", className)} aria-label={`In call: ${occupants.map((o) => o.name).join(", ")}`}>
      {occupants.map((occupant) => (
        <div key={occupant.identity} className="flex min-w-0 items-center gap-1.5 py-0.5" title={occupant.name}>
          <PersonAvatar login={lookup(occupant.identity).login} src={lookup(occupant.identity).avatar_url} className="size-4 text-[8px]" />
          <span className="min-w-0 truncate text-xs text-muted-foreground">{occupant.name}</span>
        </div>
      ))}
    </div>
  );
};
