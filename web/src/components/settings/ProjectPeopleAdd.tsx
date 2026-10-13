import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { TargetChoices } from "@/components/access/TargetChoices";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { useProjectPeopleActions } from "@/hooks/useProjectPeopleActions";
import type { PermissionGrant } from "@/models/Permission";
import { grantKey } from "@/models/PermissionPanel";
import type { Project } from "@/models/Project";

interface ProjectPeopleAddProps {
  project: Project;
  grants: PermissionGrant[];
}

// The card's footer: adding lists the role or person, which is the rule that lets it in, and opens nothing.
export const ProjectPeopleAdd = ({ project, grants }: ProjectPeopleAddProps) => {
  const [open, setOpen] = useState(false);
  const { roles, people } = usePanelTargets({ workspaceId: project.workspace_id });
  const { add, saving } = useProjectPeopleActions(project);
  const listed = grants.map(grantKey);
  const free = (t: { key: string }) => !listed.includes(t.key);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="sm" loading={saving} className="text-muted-foreground">
          <PlusIcon className="size-4" aria-hidden />
          Add a role or person
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="max-h-80 w-64 overflow-y-auto p-1">
        <TargetChoices
          roles={roles.filter(free)}
          people={people.filter(free)}
          blockedFor={() => undefined}
          onPick={(t) => {
            add(t);
            setOpen(false);
          }}
        />
      </PopoverContent>
    </Popover>
  );
};
