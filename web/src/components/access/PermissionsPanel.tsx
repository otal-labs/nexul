import { useEffect } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { CheckAccessPane } from "@/components/access/CheckAccessPane";
import { PanelTargetColumn } from "@/components/access/PanelTargetColumn";
import { RulesPane } from "@/components/access/RulesPane";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useFetchExplanation, useFetchGrants, useSaveRules } from "@/hooks/PermissionHooks";
import { usePanelEntity } from "@/hooks/usePanelEntity";
import { usePermissionPanelStore } from "@/stores/permissionPanelStore";
import type { PermissionRulesFormData } from "@/models/Permission";
import { changesOf } from "@/models/PermissionPanel";

// The dialog's body: people on one side, the chosen person's rules or answers on the other. Nothing applies until Save.
export const PermissionsPanel = () => {
  const entity = usePanelEntity();
  const { onSubmit, setLoading } = useFormDialogContext<PermissionRulesFormData>();
  const mode = usePermissionPanelStore((s) => s.mode);
  const grants = useFetchGrants(entity.type, entity.id);
  const mine = useFetchExplanation(entity.type, entity.id);
  const saveRules = useSaveRules();
  const ready = !!grants.data && !!mine.data;

  useEffect(() => {
    setLoading(!ready);
  }, [ready, setLoading]);

  onSubmit(async (values) => {
    await saveRules.mutateAsync({ resourceType: entity.type, resourceId: entity.id, changes: changesOf(values.rules, grants.data ?? []) });
    return values;
  });

  return (
    <div className="@container/panel -mx-6 -my-1 flex min-h-0 flex-1 flex-col border-t border-border">
      {(grants.isPending || mine.isPending) && <LoadingDisplay />}
      {grants.error && <ErrorDisplay error={grants.error} />}
      {!grants.error && mine.error && <ErrorDisplay error={mine.error} />}
      {ready && (
        <div className="flex min-h-0 flex-1 flex-col gap-4 px-6 pt-4 @3xl/panel:flex-row @3xl/panel:gap-0 @3xl/panel:pt-0 @3xl/panel:pl-3">
          <PanelTargetColumn />
          <div className="min-h-0 flex-1 overflow-y-auto pb-2 @3xl/panel:py-5 @3xl/panel:pl-6">
            {mode === "rules" && <RulesPane />}
            {mode === "check" && <CheckAccessPane />}
          </div>
        </div>
      )}
    </div>
  );
};
