import { ActionButton } from "@/components/action/ActionButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { TemplateResetButton } from "@/components/templates/TemplateOriginLine";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import type { TemplateKind } from "@/models/Template";

interface WorkspaceTemplateHeadProps {
  kind: TemplateKind;
  name: string;
  workspaceId: string;
  edited: boolean;
  description: string;
  // The permission a clone needs: the one the card's own Save needs.
  clonePermission: "memories:template" | "workspaces:write";
}

// A workspace template card's head: where its text comes from, said once above what it does, with Clone to… on the right.
export const useWorkspaceTemplateHead = ({ kind, name, workspaceId, edited, description, clonePermission }: WorkspaceTemplateHeadProps) => {
  const openClone = useCloneTemplateDialog();
  const at = { scope: "workspace" as const, workspace_id: workspaceId };
  return {
    description: (
      <span className="flex flex-col gap-2">
        <span className="flex items-center gap-3">
          <SettingsStatus tone={edited ? "info" : "muted"}>{edited ? "Edited here" : "Instance template"}</SettingsStatus>
          {edited && (
            <TemplateResetButton kind={kind} templateKey="" at={at} className="text-foreground underline decoration-muted-foreground/50 underline-offset-4 hover:decoration-foreground" />
          )}
        </span>
        <span>{description}</span>
      </span>
    ),
    aside: (
      <ActionButton
        permission={clonePermission}
        verb="clone this template"
        type="button"
        variant="outline"
        size="sm"
        className="-mt-1"
        onClick={() => void openClone({ kind, key: "", name, from: at })}
      >
        Clone to…
      </ActionButton>
    ),
  };
};
