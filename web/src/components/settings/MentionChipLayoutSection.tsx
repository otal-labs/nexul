import { useState, type FormEvent } from "react";

import { MentionChipField } from "@/components/settings/MentionChipField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { TemplateOriginLine } from "@/components/templates/TemplateOriginLine";
import { Button } from "@/components/ui/button";
import { useCloneTemplateDialog } from "@/hooks/useCloneTemplateDialog";
import { useFlash } from "@/hooks/useFlash";
import { useUpdateMentionChipTemplate } from "@/hooks/WorkspaceHooks";
import type { Workspace } from "@/models/Workspace";

interface MentionChipLayoutSectionProps {
  workspace: Workspace;
}

// Gated on workspaces:write, same gate-in-parent pattern as RoleSettingsSection.
const FORM_ID = "mention-chip-form";

export const MentionChipLayoutSection = ({ workspace }: MentionChipLayoutSectionProps) => {
  const [template, setTemplate] = useState(workspace.mention_chip_template);
  const updateTemplate = useUpdateMentionChipTemplate();
  const openClone = useCloneTemplateDialog();
  const [saved, flash] = useFlash();
  const at = { scope: "workspace" as const, workspace_id: workspace.id };

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await updateTemplate.mutateAsync({ id: workspace.id, template });
    flash();
  };

  return (
    <SettingsCard
      id="mention-layout"
      title="Mention chip layout"
      description="What a ticket mention shows in this workspace. The icon is fixed; the rest comes from this format."
      aside={
        <TemplateOriginLine
          kind="mention_chip"
          templateKey=""
          at={at}
          state={workspace.mention_chip_template_edited ? "edited" : "following"}
          canReset
        />
      }
      footer={
        <SettingsSaveBar
          form={FORM_ID}
          dirty={template !== workspace.mention_chip_template}
          saving={updateTemplate.isPending}
          saved={saved}
          onDiscard={() => setTemplate(workspace.mention_chip_template)}
        />
      }
    >
      <form id={FORM_ID} onSubmit={(event) => void onSubmit(event).catch(() => undefined)} className="space-y-3">
        <MentionChipField id="mention-chip-template" value={template} onChange={setTemplate} />
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="-ml-2.5"
          onClick={() => void openClone({ kind: "mention_chip", key: "", name: "Mention chip", from: at })}
        >
          Clone to…
        </Button>
      </form>
    </SettingsCard>
  );
};
