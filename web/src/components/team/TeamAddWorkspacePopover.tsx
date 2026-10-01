import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { TeamRoleSelect } from "@/components/team/TeamRoleSelect";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import type { TeamWorkspace } from "@/models/Team";
import { membershipIn } from "@/utils/TeamUtility";

interface TeamAddWorkspacePopoverProps {
  workspaces: TeamWorkspace[];
  onAdded: (workspaceId: string) => void;
}

// Offers only workspaces the person is not in, the viewer manages, and that have a role to give; the add waits for Confirm.
export const TeamAddWorkspacePopover = ({ workspaces, onAdded }: TeamAddWorkspacePopoverProps) => {
  const { person, draft, dispatch } = useMemberDraft();
  const [open, setOpen] = useState(false);
  const [pickedWorkspace, setPickedWorkspace] = useState("");
  const [pickedRole, setPickedRole] = useState("");
  const candidates = workspaces.filter(
    (workspace) => workspace.can_manage_members && !membershipIn(person, workspace.id) && !draft[workspace.id]?.added && workspace.roles.some((role) => !role.is_owner),
  );
  const workspace = candidates.find((candidate) => candidate.id === pickedWorkspace) ?? candidates[0];
  const assignable = workspace?.roles.filter((role) => !role.is_owner) ?? [];
  const roleId = assignable.find((role) => role.id === pickedRole)?.id ?? assignable[0]?.id ?? "";
  if (person.status === "removed" || !workspace) return null;

  const add = () => {
    dispatch({ type: "add", workspaceId: workspace.id, roleId });
    onAdded(workspace.id);
    setPickedWorkspace("");
    setPickedRole("");
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
        <div className="grid gap-2">
          <Select value={workspace.id} onValueChange={(id) => { setPickedWorkspace(id); setPickedRole(""); }}>
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
          <TeamRoleSelect label={`Role to add in ${workspace.name}`} roles={workspace.roles} value={roleId} onChange={setPickedRole} />
        </div>
        <Button type="button" size="sm" className="w-full" onClick={add}>
          Add
        </Button>
      </PopoverContent>
    </Popover>
  );
};
