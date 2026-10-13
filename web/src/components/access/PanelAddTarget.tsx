import { useState } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { TargetChoices } from "@/components/access/TargetChoices";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePanelTargets } from "@/hooks/usePanelTargets";
import { usePermissionsDialog } from "@/hooks/usePermissionsDialog";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { projectEntity, targetsProject } from "@/models/PermissionPanel";

interface PanelAddTargetProps {
  listed: string[];
}

// At the target column's foot: the roles and members not listed yet; a doc or folder refuses one who isn't in its project, and
// to whoever manages that project offers its list, open on them.
export const PanelAddTarget = ({ listed }: PanelAddTargetProps) => {
  const entity = usePanelEntity();
  const [open, setOpen] = useState(false);
  const add = usePermissionPanelStore((s) => s.add);
  const { roles, people, blockedFor, project } = usePanelTargets({ workspaceId: entity.workspaceId, projectId: targetsProject(entity) });
  const managesProject = useCanManageAccess(project && { type: "project", id: project.id });
  const openPanel = usePermissionsDialog();
  const free = (t: { key: string }) => !listed.includes(t.key);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="ghost" size="sm" className="shrink-0 justify-start text-muted-foreground">
          <Plus aria-hidden />
          Add role or person
        </Button>
      </PopoverTrigger>
      <PopoverContent className="max-h-80 w-64 overflow-y-auto p-1">
        <TargetChoices
          roles={roles.filter(free)}
          people={people.filter(free)}
          blockedFor={blockedFor}
          onPick={(t) => {
            add(t.key);
            setOpen(false);
          }}
          onBlocked={
            project && managesProject
              ? (t) => {
                  setOpen(false);
                  void openPanel(projectEntity(project), t.key);
                }
              : undefined
          }
        />
      </PopoverContent>
    </Popover>
  );
};
