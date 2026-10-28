import { CheckIcon } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";
import type { TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { PersonAvatar } from "@/components/PersonAvatar";
import { useTicketPerson } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";

interface DeveloperStackProps {
  developers: TicketPersonRef[];
  // The keys of the developers the board is narrowed to.
  selected: string[];
  onToggle: (developer: string) => void;
}

interface DeveloperToggleProps {
  developer: TicketPersonRef;
  active: boolean;
  dimmed: boolean;
  onToggle: (developer: string) => void;
}

const MAX_VISIBLE = 3;

const DeveloperStackAvatar = ({ developer, active, dimmed, onToggle }: DeveloperToggleProps) => {
  const person = useTicketPerson(developer);
  return (
  <button
    type="button"
    aria-pressed={active}
    aria-label={`Developer ${personLabel(person)}`}
    onClick={() => onToggle(developer.key)}
    className={cn(
      "relative rounded-full ring-2 ring-card transition-[transform,opacity,box-shadow] duration-150 ease-standard hover:z-10 hover:scale-110 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-ring",
      active && "z-10 ring-primary",
      dimmed && "opacity-50 hover:opacity-100",
    )}
  >
    <PersonAvatar login={person.login} src={person.avatar_url} className="size-7 text-[11px]" />
  </button>
  );
};

const DeveloperStackMenuItem = ({ developer, active, onToggle }: Omit<DeveloperToggleProps, "dimmed">) => {
  const person = useTicketPerson(developer);
  return (
  <button
    type="button"
    aria-pressed={active}
    aria-label={`Developer ${personLabel(person)}`}
    onClick={() => onToggle(developer.key)}
    className="quiet-focus flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-accent focus-visible:bg-accent"
  >
    <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" />
    <span className="min-w-0 flex-1 truncate">{personLabel(person)}</span>
    {active && <CheckIcon className="size-3.5 shrink-0" aria-hidden />}
  </button>
  );
};

// Overlapping avatars beside the Filter button: a click narrows the board to that developer, a ring marks the selected ones.
export const DeveloperStack = ({ developers, selected, onToggle }: DeveloperStackProps) => {
  // Selected first, so picking someone from the overflow list surfaces them in the stack.
  const ordered = [...developers.filter((d) => selected.includes(d.key)), ...developers.filter((d) => !selected.includes(d.key))];
  const visible = ordered.slice(0, MAX_VISIBLE);
  const overflow = ordered.slice(MAX_VISIBLE);
  const anySelected = selected.length > 0;

  return (
    <div role="group" aria-label="Filter by developer" className="flex items-center -space-x-2">
      {visible.map((developer) => (
        <DeveloperStackAvatar
          key={developer.key}
          developer={developer}
          active={selected.includes(developer.key)}
          dimmed={anySelected && !selected.includes(developer.key)}
          onToggle={onToggle}
        />
      ))}
      {overflow.length > 0 && (
        <Popover>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label={`${overflow.length} more developers`}
              className="relative flex size-7 items-center justify-center rounded-full bg-muted font-mono text-[11px] text-muted-foreground ring-2 ring-card transition-[color,background-color] duration-150 ease-standard hover:bg-accent hover:text-foreground focus-visible:z-10 focus-visible:outline-none focus-visible:ring-ring"
            >
              +{overflow.length}
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-1">
            {overflow.map((developer) => (
              <DeveloperStackMenuItem key={developer.key} developer={developer} active={selected.includes(developer.key)} onToggle={onToggle} />
            ))}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
};
