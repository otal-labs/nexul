import { useState } from "react";
import { Plus } from "lucide-react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { EmptyRow } from "@/components/EmptyRow";
import { microheaderClass } from "@/components/Microheader";
import { menuItemClass } from "@/components/MenuItem";
import { PersonAvatar } from "@/components/PersonAvatar";
import { useFetchProjectPeople, useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { cn } from "@/lib/utils";

interface PersonChoiceProps {
  person: Person;
  // Set when the person can't be listed here yet: they aren't in the doc's project.
  blocked?: string | undefined;
  onPick: () => void;
}

const PersonChoice = ({ person, blocked, onPick }: PersonChoiceProps) => (
  <button type="button" disabled={!!blocked} onClick={onPick} className={cn(menuItemClass, "w-full")}>
    <PersonAvatar login={person.login} src={person.avatar_url} className="size-5" label={personLabel(person)} />
    <span className="min-w-0 flex-1">
      <span className="block truncate">{personLabel(person)}</span>
      {blocked && <span className="block text-xs text-muted-foreground">{blocked}</span>}
    </span>
  </button>
);

interface PanelAddPersonProps {
  listed: string[];
}

// At the target column's foot: the members not listed yet; a doc refuses someone who isn't in its project.
export const PanelAddPerson = ({ listed }: PanelAddPersonProps) => {
  const entity = usePanelEntity();
  const [open, setOpen] = useState(false);
  const add = usePermissionPanelStore((s) => s.add);
  const { data: people } = useFetchWorkspacePeople(entity.workspaceId);
  const { data: inProject } = useFetchProjectPeople(entity.projectId || undefined);
  const { data: project } = useFetchProject(entity.projectId || undefined);
  const free = (people ?? []).filter((p) => !listed.includes(p.user_id));
  const blockedFor = (person: Person) =>
    inProject && !inProject.some((p) => p.user_id === person.user_id) ? `Add ${personLabel(person)} to ${project?.name ?? "the project"} first` : undefined;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="sm" className="shrink-0 justify-start text-muted-foreground">
          <Plus aria-hidden />
          Add person
        </Button>
      </PopoverTrigger>
      <PopoverContent className="max-h-80 w-64 overflow-y-auto p-1">
        <p className={cn(microheaderClass, "px-2 pt-2 pb-1")}>People</p>
        {free.map((person) => (
          <PersonChoice
            key={person.user_id}
            person={person}
            blocked={blockedFor(person)}
            onPick={() => {
              add(person.user_id);
              setOpen(false);
            }}
          />
        ))}
        {people && free.length === 0 && <EmptyRow className="px-2 py-2">Every person is listed.</EmptyRow>}
      </PopoverContent>
    </Popover>
  );
};
