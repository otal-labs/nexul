import { PermissionsPanel } from "@/components/access/PermissionsPanel";
import { PermissionsPanelHeader } from "@/components/access/PermissionsPanelHeader";
import { useFormDialog } from "@/hooks/useFormDialog";
import { PanelEntityContext } from "@/hooks/usePanelEntity";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import { PermissionRulesFormSchema, type PermissionRulesFormData } from "@/models/Permission";
import type { PanelEntity } from "@/models/PermissionPanel";

// Opens the permissions panel on a doc or play; its rules are held until Save.
export const usePermissionsDialog = () => {
  const { open } = useFormDialog();
  return (entity: PanelEntity) => {
    usePermissionPanelStore.getState().reset();
    return open<PermissionRulesFormData>({
      title: "Permissions",
      header: (
        <PanelEntityContext.Provider value={entity}>
          <PermissionsPanelHeader />
        </PanelEntityContext.Provider>
      ),
      footerStart: <p className="text-xs text-muted-foreground">Rules apply when you save.</p>,
      form: (
        <PanelEntityContext.Provider value={entity}>
          <PermissionsPanel />
        </PanelEntityContext.Provider>
      ),
      schema: PermissionRulesFormSchema,
      formOptions: { defaultValues: { rules: {} } },
      okLabel: "Save",
      skipAutoFocus: true,
      dialogClassName: "h-[min(46rem,calc(100dvh-4rem))] sm:max-w-4xl",
    });
  };
};
