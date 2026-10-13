import type { ReactNode } from "react";

import { PermissionsPanel } from "@/components/access/PermissionsPanel";
import { PermissionsPanelHeader } from "@/components/access/PermissionsPanelHeader";
import { useFormDialog } from "@/hooks/useFormDialog";
import { PanelEntityContext } from "@/hooks/usePanelEntity";
import { createPermissionPanelStore, PermissionPanelStoreContext } from "@/stores/permissionPanelStore";
import { PermissionRulesFormSchema, type PermissionRulesFormData } from "@/models/Permission";
import type { PanelEntity } from "@/models/PermissionPanel";

// Opens the permissions panel on a doc, folder, play, project or stack, on target (a targetKey) when named; rules are held until Save.
export const usePermissionsDialog = () => {
  const { open } = useFormDialog();
  return (entity: PanelEntity, target?: string) => {
    const store = createPermissionPanelStore(target ?? null);
    const scoped = (children: ReactNode) => (
      <PanelEntityContext.Provider value={entity}>
        <PermissionPanelStoreContext.Provider value={store}>{children}</PermissionPanelStoreContext.Provider>
      </PanelEntityContext.Provider>
    );
    return open<PermissionRulesFormData>({
      title: "Permissions",
      header: scoped(<PermissionsPanelHeader />),
      footerStart: <p className="text-xs text-muted-foreground">Rules apply when you save.</p>,
      form: scoped(<PermissionsPanel />),
      schema: PermissionRulesFormSchema,
      formOptions: { defaultValues: { rules: {} } },
      okLabel: "Save",
      skipAutoFocus: true,
      dialogClassName: "h-[min(46rem,calc(100dvh-4rem))] sm:max-w-4xl",
    });
  };
};
