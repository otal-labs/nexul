import { CheckIcon, PlusIcon } from "lucide-react";

import { PopoverContent } from "@/components/ui/popover";
import type { Workspace } from "@/models/Workspace";

const menuItemClass =
  "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13.5px] text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:bg-accent/60 focus-visible:text-foreground";

interface WorkspaceSwitcherMenuProps {
  workspaces: Workspace[] | undefined;
  selectedWorkspaceId: string;
  onSelect: (workspaceId: string) => void;
  canCreateWorkspace: boolean;
  onCreate: () => void;
}

export const WorkspaceSwitcherMenu = ({
  workspaces,
  selectedWorkspaceId,
  onSelect,
  canCreateWorkspace,
  onCreate,
}: WorkspaceSwitcherMenuProps) => (
  <PopoverContent side="bottom" align="start" sideOffset={6} className="w-56 p-1.5">
    <div className="flex flex-col gap-0.5">
      {workspaces?.map((ws) => (
        <button key={ws.id} type="button" onClick={() => onSelect(ws.id)} className={menuItemClass}>
          <span className="flex size-5 shrink-0 items-center justify-center rounded bg-accent text-[10px] font-semibold text-primary">
            {ws.name[0]}
          </span>
          <span className="flex-1 truncate">{ws.name}</span>
          {ws.id === selectedWorkspaceId && <CheckIcon className="size-3.5 shrink-0 text-primary" aria-hidden />}
        </button>
      ))}
      {canCreateWorkspace && (
        <>
          <div className="my-1 border-t border-border" />
          <button type="button" onClick={onCreate} className={menuItemClass}>
            <PlusIcon className="size-4 shrink-0" aria-hidden />
            <span>New Workspace</span>
          </button>
        </>
      )}
    </div>
  </PopoverContent>
);
