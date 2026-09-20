import { SettingsSectionNav, type SettingsSectionNavItem } from "@/components/settings/SettingsSectionNav";

export const SETTINGS_SECTIONS = [
  "instance",
  "roles",
  "plays",
  "mentions",
  "appearance",
  "tokens",
  "pairing",
  "connectors",
  "dns",
  "automation-secrets",
  "access",
  "danger",
] as const;

export type SettingsSection = (typeof SETTINGS_SECTIONS)[number];

export const DEFAULT_SETTINGS_SECTION: SettingsSection = "instance";

export const isSettingsSection = (value: string | null | undefined): value is SettingsSection =>
  !!value && (SETTINGS_SECTIONS as readonly string[]).includes(value);

const sectionLabels: Record<SettingsSection, string> = {
  instance: "Instance",
  roles: "Roles",
  plays: "Plays",
  mentions: "Mention chips",
  appearance: "Appearance",
  tokens: "Tokens",
  pairing: "T3 pairing",
  connectors: "Connectors",
  dns: "DNS",
  "automation-secrets": "Automation secrets",
  access: "Instance access",
  danger: "Danger zone",
};

interface SettingsNavProps {
  active: SettingsSection;
  // AllowlistSection only renders for admins — the nav must not link to an empty section.
  showInstanceAccess: boolean;
  // RoleSettingsSection only renders for roles:write holders — same "no empty link" rule.
  showRoles: boolean;
  // PlaySettingsSection only renders for plays:read holders — same "no empty link" rule.
  showPlays: boolean;
  // MentionChipLayoutSection gates on workspaces:write.
  showMentionLayout: boolean;
}

export const SettingsNav = ({
  active,
  showInstanceAccess,
  showRoles,
  showPlays,
  showMentionLayout,
}: SettingsNavProps) => {
  const items: SettingsSectionNavItem[] = SETTINGS_SECTIONS.filter((section) => {
    if (section === "roles") return showRoles;
    if (section === "plays") return showPlays;
    if (section === "mentions") return showMentionLayout;
    if (section === "access") return showInstanceAccess;
    return true;
  }).map((section) => ({ section, label: sectionLabels[section], danger: section === "danger" }));

  return <SettingsSectionNav ariaLabel="Settings sections" active={active} items={items} />;
};
