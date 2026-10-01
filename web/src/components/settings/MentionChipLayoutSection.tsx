import { useState, type FormEvent } from "react";

import { MentionChipField } from "@/components/settings/MentionChipField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useUpdateMentionChipTemplate } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";

interface MentionChipLayoutSectionProps {
  workspace: Workspace;
}

// Gated on workspaces:write, same gate-in-parent pattern as RoleSettingsSection.
export const MentionChipLayoutSection = ({ workspace }: MentionChipLayoutSectionProps) => {
  const [template, setTemplate] = useState(workspace.mention_chip_template);
  const updateTemplate = useUpdateMentionChipTemplate();
  const openClone = useCloneTemplateDialog();
  const at = { scope: "workspace" as const, workspace_id: workspace.id };

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await updateTemplate.mutateAsync({ id: workspace.id, template });
  };

  return (
    <SettingsCard
      id="mention-layout"
      title="Mention chip layout"
      description="What a @-mention ticket chip shows in this workspace. The icon stays fixed — everything else comes from this format string."
    >
      <form onSubmit={onSubmit} className="space-y-4">
        <TemplateOriginLine
          kind="mention_chip"
          templateKey=""
          at={at}
          state={workspace.mention_chip_template_edited ? "edited" : "following"}
          canReset
        />
        <MentionChipField id="mention-chip-template" value={template} onChange={setTemplate} />
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => void openClone({ kind: "mention_chip", key: "", name: "Mention chip", from: at })}
          >
            Clone to…
          </Button>
          <Button type="submit" size="sm" loading={updateTemplate.isPending} disabled={template === workspace.mention_chip_template}>
            Save
          </Button>
        </div>
      </form>
    </SettingsCard>
  );
};
