import { UserPlus } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PersonPicker } from "@/components/accessgrid/PersonPicker";
import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface PeopleCellProps {
  projectName: string;
  userIds: string[];
  onToggle: (userId: string) => void;
}

const SHOWN = 3;

// The people a project names beyond its roles as faces, and the add button, which shows on the row's hover or focus.
export const PeopleCell = ({ projectName, userIds, onToggle }: PeopleCellProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const people = userIds.map(lookup);
  const names = people.map(personLabel).join(", ");
  return (
    <span className="flex shrink-0 items-center gap-1">
      {people.length > 0 && (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="flex -space-x-1" aria-label={names}>
              {people.slice(0, SHOWN).map((person) => (
                <PersonAvatar key={person.user_id} login={person.login} name={personLabel(person)} src={person.avatar_url} label={personLabel(person)} className="size-5 text-[10px] ring-2 ring-card" />
              ))}
              {people.length > SHOWN && (
                <span className="avatar-more size-5 text-[10px] ring-2 ring-card">+{people.length - SHOWN}</span>
              )}
            </span>
          </TooltipTrigger>
          <TooltipContent>{names}</TooltipContent>
        </Tooltip>
      )}
      <Popover>
        <Tooltip>
          <TooltipTrigger asChild>
            <PopoverTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 data-[state=open]:opacity-100"
                aria-label={`Add a person to ${projectName}`}
              >
                <UserPlus className="size-4" aria-hidden />
              </Button>
            </PopoverTrigger>
          </TooltipTrigger>
          <TooltipContent>Add a person</TooltipContent>
        </Tooltip>
        <PopoverContent align="end" className="w-72 p-1">
          <PersonPicker named={userIds} onToggle={onToggle} />
        </PopoverContent>
      </Popover>
    </span>
  );
};
