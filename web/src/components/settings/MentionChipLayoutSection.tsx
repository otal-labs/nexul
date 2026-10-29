import { useState, type FormEvent } from "react";

import { MentionChipField } from "@/components/settings/MentionChipField";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { useWorkspaceTemplateHead } from "@/components/settings/WorkspaceTemplateHead";
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
  const [saved, flash] = useFlash();
  const head = useWorkspaceTemplateHead({
    kind: "mention_chip",
    name: "Mention chip",
    workspaceId: workspace.id,
    edited: workspace.mention_chip_template_edited,
    description: "What a ticket mention shows in chat and docs.",
    clonePermission: "workspaces:write",
  });

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault();
    await updateTemplate.mutateAsync({ id: workspace.id, template });
    flash();
  };

  return (
    <SettingsCard
      id="mention-layout"
      title="Ticket mentions"
      description={head.description}
      aside={head.aside}
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
      </form>
    </SettingsCard>
  );
};
