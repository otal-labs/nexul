import { AccountsSection } from "@/components/settings/AccountsSection";
import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { InstanceSettingsPanel } from "@/components/settings/InstanceSettingsPanel";
import { MembersSection } from "@/components/settings/MembersSection";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { OAuthProviderSection } from "@/components/settings/OAuthProviderSection";
import { PlaySettingsSection } from "@/components/settings/PlaySettingsSection";
import { RoleSettingsSection } from "@/components/settings/RoleSettingsSection";
import type { SettingsSection } from "@/components/settings/SettingsNav";
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

const MembersPanel = ({ canManageMembers }: { canManageMembers: boolean }) => (
  <>{canManageMembers && <MembersSection />}</>
);

const MentionsPanel = ({
  settings,
  canManageMentionLayout,
}: {
  settings: InstanceSettings | undefined;
  canManageMentionLayout: boolean;
}) => <>{settings && canManageMentionLayout && <MentionChipLayoutSection settings={settings} />}</>;

const InstancePanel = ({ settings, isInstanceAdmin }: { settings: InstanceSettings | undefined; isInstanceAdmin: boolean }) => (
  <>{settings && isInstanceAdmin && <InstanceSettingsPanel settings={settings} />}</>
);

const SignInPanel = ({ settings, isInstanceAdmin }: { settings: InstanceSettings | undefined; isInstanceAdmin: boolean }) => (
  <>
    {settings && isInstanceAdmin && <OAuthProviderSection provider="google" settings={settings} />}
    {settings && isInstanceAdmin && <OAuthProviderSection provider="discord" settings={settings} />}
  </>
);

const ConnectorsPanel = ({ isInstanceAdmin }: { isInstanceAdmin: boolean }) => (
  <>{isInstanceAdmin && <ConnectorsSettingsPanel />}</>
);

const DnsPanel = ({ isInstanceAdmin }: { isInstanceAdmin: boolean }) => <>{isInstanceAdmin && <GatewaysSection />}</>;

const AccessPanel = ({ isInstanceAdmin }: { isInstanceAdmin: boolean }) => (
  <>{isInstanceAdmin && <AccountsSection />}</>
);

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
  canManageMembers: boolean;
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
  canManageMembers,
  canManageMentionLayout,
}: SettingsPageContentProps) => (
  <>
    {section === "roles" && <RolesPanel canManageRoles={canManageRoles} />}
    {section === "plays" && (
      <PlaysPanel canReadPlays={canReadPlays} canWritePlays={canWritePlays} canDeletePlays={canDeletePlays} />
    )}
    {section === "interview" && <InterviewTemplateSection />}
    {section === "members" && <MembersPanel canManageMembers={canManageMembers} />}
    {section === "mentions" && (
      <MentionsPanel settings={settings} canManageMentionLayout={canManageMentionLayout} />
    )}
    {section === "danger" && <DangerPanel settings={settings} />}
    {section === "instance" && <InstancePanel settings={settings} isInstanceAdmin={isInstanceAdmin} />}
    {section === "sign-in" && <SignInPanel settings={settings} isInstanceAdmin={isInstanceAdmin} />}
    {section === "connectors" && <ConnectorsPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "dns" && <DnsPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "access" && <AccessPanel isInstanceAdmin={isInstanceAdmin} />}
  </>
);
