import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
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

const RolesPanel = ({ canManageRoles }: { canManageRoles: boolean }) => (
  <>{canManageRoles && <RoleSettingsSection />}</>
);

const PlaysPanel = ({ canReadPlays, canWritePlays, canDeletePlays }: {
  canReadPlays: boolean;
  canWritePlays: boolean;
  canDeletePlays: boolean;
}) => <>{canReadPlays && <PlaySettingsSection canWrite={canWritePlays} canDelete={canDeletePlays} />}</>;

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
  canManageRoles: boolean;
  canReadPlays: boolean;
  canWritePlays: boolean;
  canDeletePlays: boolean;
}

// One card per workspace section (mirrors ProjectSettingsPage); the instance sections are InstanceSettingsContent.
export const SettingsPageContent = ({
  section,
  canManageWorkspace,
  canManageRoles,
  canReadPlays,
  canWritePlays,
  canDeletePlays,
}: SettingsPageContentProps) => (
  <>
    {section === "general" && <GeneralPanel canManageWorkspace={canManageWorkspace} />}
    {section === "roles" && <RolesPanel canManageRoles={canManageRoles} />}
    {section === "plays" && (
      <PlaysPanel canReadPlays={canReadPlays} canWritePlays={canWritePlays} canDeletePlays={canDeletePlays} />
    )}
    {section === "interview" && <InterviewTemplateSection />}
    {section === "mentions" && <MentionsPanel canManageWorkspace={canManageWorkspace} />}
    {section === "danger" && <DangerZoneSection />}
    {section === "team" && <TeamSection />}
  </>
);
