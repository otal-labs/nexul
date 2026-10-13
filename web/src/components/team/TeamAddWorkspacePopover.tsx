import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import type { TeamWorkspace } from "@/models/Team";
import { membershipIn } from "@/utils/TeamUtility";

interface TeamAddWorkspacePopoverProps {
  workspaces: TeamWorkspace[];
  onAdded: (workspaceId: string) => void;
}

// Offers only workspaces the person is not in, the viewer manages, and that have a role to give; the new tab picks the roles.
export const TeamAddWorkspacePopover = ({ workspaces, onAdded }: TeamAddWorkspacePopoverProps) => {
  const { person, draft, dispatch } = useMemberDraft();
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState("");
  const candidates = workspaces.filter(
    (workspace) => workspace.can_manage_members && !membershipIn(person, workspace.id) && !draft[workspace.id]?.added && workspace.roles.some((role) => !role.is_owner),
  );
  const workspace = candidates.find((candidate) => candidate.id === picked) ?? candidates[0];
  if (person.status === "removed" || !workspace) return null;

  const add = () => {
    dispatch({ type: "add", workspaceId: workspace.id });
    onAdded(workspace.id);
    setPicked("");
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="ghost" size="icon" aria-label="Add to a workspace" title="Add to a workspace" className="mb-1 size-7 shrink-0 text-muted-foreground">
          <PlusIcon className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-72 space-y-3">
        <p className="text-sm font-medium">Add to a workspace</p>
        <Select value={workspace.id} onValueChange={setPicked}>
          <SelectTrigger aria-label="Workspace to add to" className="h-8 w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {candidates.map((candidate) => (
              <SelectItem key={candidate.id} value={candidate.id}>
                {candidate.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" size="sm" className="w-full" onClick={add}>
          Add
        </Button>
      </PopoverContent>
    </Popover>
  );
};
