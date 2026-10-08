import { PersonAvatar } from "@/components/PersonAvatar";
import { useProjectAreas } from "@/hooks/PermissionHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { accessSummary } from "@/models/PermissionLevel";
import type { ProjectAccessEntry } from "@/models/Project";

interface ProjectPeopleAccessRowProps {
  entry: ProjectAccessEntry;
}

export const ProjectPeopleAccessRow = ({ entry }: ProjectPeopleAccessRowProps) => {
  const person = usePerson(entry.user_id);
  const areas = useProjectAreas();
  return (
    <li className="flex min-h-11 items-center gap-3 py-2">
      <PersonAvatar login={person.login} src={person.avatar_url} label={entry.name} className="size-7 text-xs" />
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="truncate text-sm">{entry.name}</p>
        <p className="truncate font-mono text-xs text-muted-foreground">{accessSummary(areas, entry.actions)}</p>
      </div>
    </li>
  );
};
