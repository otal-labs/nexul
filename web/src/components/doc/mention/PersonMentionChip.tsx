import { PersonAvatar } from "@/components/PersonAvatar";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { personLabel } from "@/models/Person";
import { cn } from "@/lib/utils";

interface PersonMentionChipProps {
  id: string;
  // The login stored at insert time, shown until People loads.
  label: string;
}

// Read live from People, so a rename or a new picture shows at once; it opens nothing when clicked.
export const PersonMentionChip = ({ id, label }: PersonMentionChipProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: people } = useFetchWorkspacePeople(workspaceId || undefined);
  const person = people?.find((p) => p.user_id === id);
  const unknown = !!people && !person;
  const fallback = unknown ? "@unknown" : `@${label}`;

  return (
    <span
      className={cn("mention-chip", "mention-chip--inert", unknown && "text-muted-foreground")}
      data-testid="mention-chip"
      data-mention-type="person"
      data-mention-id={id}
    >
      {person && (
        <span className="size-[1.1em] shrink-0">
          <PersonAvatar login={person.login} src={person.avatar_url} className="size-full text-[0.6em]" />
        </span>
      )}
      <span className="mention-chip__label">{person ? personLabel(person) : fallback}</span>
    </span>
  );
};
