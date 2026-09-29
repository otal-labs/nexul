import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { InstanceSettingsPanel } from "@/components/settings/InstanceSettingsPanel";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
import { SignInProvidersPanel } from "@/components/settings/SignInProvidersPanel";
import { TeamSection } from "@/components/team/TeamSection";
import type { SettingsSection } from "@/components/settings/SettingsNav";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import type { InstanceSettings } from "@/models/User";

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

const InstancePanel = ({ settings, isInstanceAdmin }: { settings: InstanceSettings | undefined; isInstanceAdmin: boolean }) => (
  <>{settings && isInstanceAdmin && <InstanceSettingsPanel settings={settings} />}</>
);

const SignInPanel = ({ settings, isInstanceAdmin }: { settings: InstanceSettings | undefined; isInstanceAdmin: boolean }) => (
  <>{settings && isInstanceAdmin && <SignInProvidersPanel settings={settings} />}</>
);

const ConnectorsPanel = ({ isInstanceAdmin }: { isInstanceAdmin: boolean }) => (
  <>{isInstanceAdmin && <ConnectorsSettingsPanel />}</>
);

const DnsPanel = ({ isInstanceAdmin }: { isInstanceAdmin: boolean }) => <>{isInstanceAdmin && <GatewaysSection />}</>;


const DangerPanel = ({ settings }: { settings: InstanceSettings | undefined }) => (
  <>{settings && <DangerZoneSection />}</>
);

interface SettingsPageContentProps {
  section: SettingsSection;
  settings: InstanceSettings | undefined;
  isInstanceAdmin: boolean;
  canManageRoles: boolean;
  canReadPlays: boolean;
  canWritePlays: boolean;
  canDeletePlays: boolean;
  canManageMentionLayout: boolean;
}

// One card per section (mirrors ProjectSettingsPage); ConfigurationPage keeps only fetching and composing this.
export const SettingsPageContent = ({
  section,
  settings,
  isInstanceAdmin,
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
    {section === "danger" && <DangerPanel settings={settings} />}
    {section === "instance" && <InstancePanel settings={settings} isInstanceAdmin={isInstanceAdmin} />}
    {section === "sign-in" && <SignInPanel settings={settings} isInstanceAdmin={isInstanceAdmin} />}
    {section === "connectors" && <ConnectorsPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "dns" && <DnsPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "team" && <TeamSection />}
  </>
);
