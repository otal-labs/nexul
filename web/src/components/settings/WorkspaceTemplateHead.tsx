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
    // Following the instance template is the default and goes unsaid; only an edit is marked.
    description: (
      <span className="flex flex-col gap-2">
        {edited && <SettingsStatus tone="info">Edited here</SettingsStatus>}
        <span>{description}</span>
      </span>
    ),
    aside: (
      <span className="-mt-1 flex items-center gap-2">
        {edited && <TemplateResetButton kind={kind} templateKey="" at={at} variant="ghost" />}
        <ActionButton
          permission={clonePermission}
          verb="clone this template"
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void openClone({ kind, key: "", name, from: at })}
        >
          Clone to…
        </ActionButton>
      </span>
    ),
  };
};
