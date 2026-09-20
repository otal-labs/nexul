import { CheckIcon } from "lucide-react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { AssigneeAvatar } from "@/components/AssigneeAvatar";
import { cn } from "@/lib/utils";

interface AssigneeStackProps {
  assignees: string[];
  selected: string[];
  onToggle: (assignee: string) => void;
}

interface AssigneeToggleProps {
  login: string;
  active: boolean;
  dimmed: boolean;
  onToggle: (assignee: string) => void;
}

const MAX_VISIBLE = 5;

const AssigneeStackAvatar = ({ login, active, dimmed, onToggle }: AssigneeToggleProps) => (
  <button
    type="button"
    aria-pressed={active}
    aria-label={`Assignee ${login}`}
    onClick={() => onToggle(login)}
    className={cn(
      "relative rounded-full ring-2 ring-card transition-[transform,opacity,box-shadow] duration-150 ease-standard hover:z-10 hover:scale-110 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-ring",
      active && "z-10 ring-primary",
      dimmed && "opacity-50 hover:opacity-100",
    )}
  >
    <AssigneeAvatar login={login} className="size-8 text-[11px]" />
  </button>
);

const AssigneeStackMenuItem = ({ login, active, onToggle }: Omit<AssigneeToggleProps, "dimmed">) => (
  <button
    type="button"
    aria-pressed={active}
    aria-label={`Assignee ${login}`}
    onClick={() => onToggle(login)}
    className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent focus-visible:bg-accent focus-visible:outline-none"
  >
    <AssigneeAvatar login={login} className="size-6" />
    <span className="min-w-0 flex-1 truncate">{login}</span>
    {active && <CheckIcon className="size-3.5 shrink-0" aria-hidden />}
  </button>
);

// Overlapping avatars beside the Filter button: a click narrows the board to that assignee, a ring marks the selected ones.
export const AssigneeStack = ({ assignees, selected, onToggle }: AssigneeStackProps) => {
  // Selected first, so picking someone from the overflow list surfaces them in the stack.
  const ordered = [...assignees.filter((a) => selected.includes(a)), ...assignees.filter((a) => !selected.includes(a))];
  const visible = ordered.slice(0, MAX_VISIBLE);
  const overflow = ordered.slice(MAX_VISIBLE);
  const anySelected = selected.length > 0;

  return (
    <div role="group" aria-label="Filter by assignee" className="flex items-center -space-x-2">
      {visible.map((login) => (
        <AssigneeStackAvatar
          key={login}
          login={login}
          active={selected.includes(login)}
          dimmed={anySelected && !selected.includes(login)}
          onToggle={onToggle}
        />
      ))}
      {overflow.length > 0 && (
        <Popover>
          <PopoverTrigger asChild>
            <button
              type="button"
              aria-label={`${overflow.length} more assignees`}
              className="relative flex size-8 items-center justify-center rounded-full bg-muted font-mono text-[10.5px] text-muted-foreground ring-2 ring-card transition-[color,background-color] duration-150 ease-standard hover:bg-accent hover:text-foreground focus-visible:z-10 focus-visible:outline-none focus-visible:ring-ring"
            >
              +{overflow.length}
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-56 p-1">
            {overflow.map((login) => (
              <AssigneeStackMenuItem key={login} login={login} active={selected.includes(login)} onToggle={onToggle} />
            ))}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
};
