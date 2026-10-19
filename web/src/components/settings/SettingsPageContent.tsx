import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { WorkspaceGeneralSection } from "@/components/settings/WorkspaceGeneralSection";
import { TeamSection } from "@/components/team/TeamSection";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";

// Thin same-file gates: each permission check gets its own function scope, not a wall of chained &&.
// Keyed by workspace id so switching workspaces reseeds the form instead of keeping the previous name.
const GeneralPanel = ({ canManageWorkspace }: { canManageWorkspace: boolean }) => {
  const workspace = useSelectedWorkspace();
  return <>{workspace && canManageWorkspace && <WorkspaceGeneralSection key={workspace.id} workspace={workspace} />}</>;
};

// Keyed by workspace and text so a switch or an instance change reseeds the form instead of keeping the old template.
const MentionsPanel = ({ canManageWorkspace }: { canManageWorkspace: boolean }) => {
  const workspace = useSelectedWorkspace();
  return (
    <>
      {workspace && canManageWorkspace && <MentionChipLayoutSection key={`${workspace.id}:${workspace.mention_chip_template}`} workspace={workspace} />}
    </>
  );
};

interface SettingsPageContentProps {
  section: SettingsSection;
  canManageWorkspace: boolean;
}

// One card per workspace section (mirrors ProjectSettingsPage); the instance sections are InstanceSettingsContent.
export const SettingsPageContent = ({ section, canManageWorkspace }: SettingsPageContentProps) => (
  <>
    {section === "general" && <GeneralPanel canManageWorkspace={canManageWorkspace} />}
    {section === "interview" && <InterviewTemplateSection />}
    {section === "mentions" && <MentionsPanel canManageWorkspace={canManageWorkspace} />}
    {section === "danger" && <DangerZoneSection />}
    {section === "team" && <TeamSection />}
  </>
);
