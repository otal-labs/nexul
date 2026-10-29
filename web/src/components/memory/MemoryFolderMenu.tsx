import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { memoryFolderOptions, type MemoryFolderId } from "@/components/memory/memoryGroups";
import { useSetMemoryFlag } from "@/hooks/MemoryHooks";
import { useActionState } from "@/hooks/useActionState";
import type { Memory } from "@/models/Memory";

interface MemoryFolderMenuProps {
  memory: Memory;
}

// The interview and the decisions log are always Main, so only an ordinary memory moves.
export const MemoryFolderMenu = ({ memory }: MemoryFolderMenuProps) => {
  const setFooter = useSetMemoryFlag("footer");
  const canMove = useActionState({ permission: "memories:write", projectId: memory.project_id }).state === "live" && memory.kind === "";
  const current: MemoryFolderId = memory.footer ? "footer" : "main";
  const folder = memoryFolderOptions.find((o) => o.id === current);
  const label = folder?.name;

  return (
    <>
      {!canMove && (
        <span className="text-foreground" title={folder?.hint}>
          {label}
        </span>
      )}
      {canMove && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="-mx-2 h-7 gap-1 px-2 text-[13px] font-normal text-foreground has-[>svg]:px-2" aria-label={`Folder: ${label}`}>
              {label}
              <ChevronDownIcon className="size-3.5 text-muted-foreground" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-60">
            <DropdownMenuRadioGroup value={current} onValueChange={(id) => setFooter.mutate({ memory, value: id === "footer" })}>
              {memoryFolderOptions.map((option) => (
                <DropdownMenuRadioItem key={option.id} value={option.id} className="items-start [&>span:first-child]:top-2">
                  <span className="flex flex-col">
                    <span>{option.name}</span>
                    <span className="text-xs text-muted-foreground">{option.hint}</span>
                  </span>
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </>
  );
};
