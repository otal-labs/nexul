import { DialogDescription } from "@/components/ui/dialog";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useFetchProject } from "@/hooks/ProjectHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionPanelStore, type PermissionPanelMode } from "@/stores/permissionPanelStore";
import { entityWord, targetsProject } from "@/models/PermissionPanel";

// "doc Pricing in Storefront › Sales" or "folder Sales in Storefront": the entity and where it sits, then Rules | Check access, a view held in the dialog.
export const PermissionsPanelHeader = () => {
  const entity = usePanelEntity();
  const mode = usePermissionPanelStore((s) => s.mode);
  const setMode = usePermissionPanelStore((s) => s.setMode);
  const { data: project } = useFetchProject(targetsProject(entity) || undefined);
  const { data: folders } = useFetchDocFolders(entity.type === "doc" ? entity.projectId : "");
  const where = [project?.name, folders?.find((f) => f.id === entity.folderId)?.name].filter(Boolean).join(" › ");

  return (
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div className="min-w-0">
        <p aria-hidden className="text-base leading-snug font-semibold">
          Permissions
        </p>
        <DialogDescription>
          {entityWord(entity.type)} <span className="font-medium text-foreground">{entity.name}</span>
          {where && ` in ${where}`}
        </DialogDescription>
      </div>
      <ToggleGroup
        type="single"
        variant="segmented"
        size="xs"
        value={mode}
        aria-label="Permissions view"
        onValueChange={(v) => v && setMode(v as PermissionPanelMode)}
      >
        <ToggleGroupItem value="rules">Rules</ToggleGroupItem>
        <ToggleGroupItem value="check">Check access</ToggleGroupItem>
      </ToggleGroup>
    </div>
  );
};
