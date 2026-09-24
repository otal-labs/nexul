import { AccountsSection } from "@/components/settings/AccountsSection";
import { AppearanceSection } from "@/components/settings/AppearanceSection";
import { AutomationSecretsSection } from "@/components/settings/AutomationSecretsSection";
import { ComputersSection } from "@/components/settings/ComputersSection";
import { ConnectionTokenSection } from "@/components/settings/ConnectionTokenSection";
import { ConnectorsSettingsPanel } from "@/components/settings/ConnectorsSettingsPanel";
import { DangerZoneSection } from "@/components/settings/DangerZoneSection";
import { GatewaysSection } from "@/components/dns/GatewaysSection";
import { InterviewTemplateSection } from "@/components/settings/InterviewTemplateSection";
import { InstanceSettingsPanel } from "@/components/settings/InstanceSettingsPanel";
import { McpConfigSection } from "@/components/settings/McpConfigSection";
import { MentionChipLayoutSection } from "@/components/settings/MentionChipLayoutSection";
import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";
import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";
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

const MentionsPanel = ({
  settings,
  canManageMentionLayout,
}: {
  settings: InstanceSettings | undefined;
  canManageMentionLayout: boolean;
}) => <>{settings && canManageMentionLayout && <MentionChipLayoutSection settings={settings} />}</>;

const TokensPanel = ({ settings }: { settings: InstanceSettings | undefined }) => (
  <>
    {settings && (
      <>
        {/* Remounts on settings_version bump so a revealed token from before a URL change never lingers. */}
        <ConnectionTokenSection key={settings.settings_version} />
        <PersonalAccessTokensSection />
      </>
    )}
  </>
);

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
  canManageMentionLayout: boolean;
}

// One card per section (mirrors ProjectSettingsPage); SettingsPage keeps only fetching and composing this.
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
    {section === "instance" && settings && (
      <InstanceSettingsPanel settings={settings} isInstanceAdmin={isInstanceAdmin} />
    )}
    {section === "roles" && <RolesPanel canManageRoles={canManageRoles} />}
    {section === "plays" && (
      <PlaysPanel canReadPlays={canReadPlays} canWritePlays={canWritePlays} canDeletePlays={canDeletePlays} />
    )}
    {section === "interview" && <InterviewTemplateSection />}
    {section === "mentions" && (
      <MentionsPanel settings={settings} canManageMentionLayout={canManageMentionLayout} />
    )}
    {section === "appearance" && <AppearanceSection />}
    {section === "tokens" && <TokensPanel settings={settings} />}
    {section === "pairing" && (
      <>
        <ComputersSection />
        <PairingDefaultsSection />
        <McpConfigSection />
      </>
    )}
    {section === "connectors" && <ConnectorsSettingsPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "dns" && <GatewaysSection />}
    {section === "automation-secrets" && <AutomationSecretsSection />}
    {section === "access" && <AccessPanel isInstanceAdmin={isInstanceAdmin} />}
    {section === "danger" && <DangerPanel settings={settings} />}
  </>
);
