import { ActionButton } from "@/components/action/ActionButton";
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

// A workspace template card's head: what it does, then where its text comes from as a meta line, with Clone to… the one action on the right.
export const useWorkspaceTemplateHead = ({ kind, name, workspaceId, edited, description, clonePermission }: WorkspaceTemplateHeadProps) => {
  const openClone = useCloneTemplateDialog();
  const at = { scope: "workspace" as const, workspace_id: workspaceId };
  return {
    // Following the instance template is the default and goes unsaid; only an edit is marked.
    description: (
      <span className="flex flex-col gap-1">
        <span>{description}</span>
        {edited && (
          <span className="flex flex-wrap items-center gap-x-3">
            <span>Edited here</span>
            <TemplateResetButton kind={kind} templateKey="" at={at} className="text-foreground underline decoration-muted-foreground/50" />
          </span>
        )}
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
