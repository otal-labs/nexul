import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
import { TeamSection } from "@/components/team/TeamSection";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";

// Thin same-file gates: each permission check gets its own function scope, not a wall of chained &&.
const RolesPanel = ({ canManageRoles }: { canManageRoles: boolean }) => (
  <>{canManageRoles && <RoleSettingsSection />}</>
);

const PlaysPanel = ({ canReadPlays, canWritePlays, canDeletePlays }: {
  canReadPlays: boolean;
  canWritePlays: boolean;
  canDeletePlays: boolean;
}) => <>{canReadPlays && <PlaySettingsSection canWrite={canWritePlays} canDelete={canDeletePlays} />}</>;

// Keyed by workspace id so switching workspaces reseeds the form instead of keeping the previous template.
const MentionsPanel = ({ canManageMentionLayout }: { canManageMentionLayout: boolean }) => {
  const workspace = useSelectedWorkspace();
  return (
    <>
      {workspace && canManageMentionLayout && <MentionChipLayoutSection key={workspace.id} workspace={workspace} />}
    </>
  );
};

interface SettingsPageContentProps {
  section: SettingsSection;
  canManageRoles: boolean;
  canReadPlays: boolean;
  canWritePlays: boolean;
  canDeletePlays: boolean;
  canManageMentionLayout: boolean;
}

// One card per workspace section (mirrors ProjectSettingsPage); the instance sections are InstanceSettingsContent.
export const SettingsPageContent = ({
  section,
  canManageRoles,
  canReadPlays,
  canWritePlays,
  canDeletePlays,
  canManageMentionLayout,
}: SettingsPageContentProps) => (
  <>
    {section === "roles" && <RolesPanel canManageRoles={canManageRoles} />}
    {section === "plays" && (
      <PlaysPanel canReadPlays={canReadPlays} canWritePlays={canWritePlays} canDeletePlays={canDeletePlays} />
    )}
    {section === "interview" && <InterviewTemplateSection />}
    {section === "mentions" && <MentionsPanel canManageMentionLayout={canManageMentionLayout} />}
    {section === "danger" && <DangerZoneSection />}
    {section === "team" && <TeamSection />}
  </>
);
