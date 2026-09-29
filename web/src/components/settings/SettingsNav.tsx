import { SettingsSectionNav, type SettingsSectionNavItem } from "@/components/settings/SettingsSectionNav";

export const SETTINGS_SECTIONS = [
  "roles",
  "plays",
  "interview",
  "mentions",
  "danger",
  "instance",
  "team",
  "sign-in",
  "connectors",
  "dns",
] as const;

export type SettingsSection = (typeof SETTINGS_SECTIONS)[number];

export const isSettingsSection = (value: string | null | undefined): value is SettingsSection =>
  !!value && (SETTINGS_SECTIONS as readonly string[]).includes(value);

const WORKSPACE_GROUP = "This workspace";
const INSTANCE_GROUP = "Whole instance";

const INSTANCE_SECTIONS: readonly SettingsSection[] = ["instance", "team", "sign-in", "connectors", "dns"];

const sectionLabels: Record<SettingsSection, string> = {
  roles: "Roles",
  plays: "Plays",
  interview: "Interview template",
  mentions: "Mention chips",
  danger: "Danger zone",
  instance: "Instance",
  "sign-in": "Sign-in providers",
  connectors: "Connectors",
  dns: "DNS",
  team: "Team",
};

export interface SettingsVisibility {
  // Every whole-instance section is for instance admins only.
  isInstanceAdmin: boolean;
  // Each gated workspace section renders only for its permission holder; the nav must not link to an empty section.
  showRoles: boolean;
  showPlays: boolean;
  showInterviewTemplate: boolean;
  showMentionLayout: boolean;
}

// The sections the viewer may open, in nav order; the page falls back to the first when the URL names none of them.
export const visibleSettingsSections = (visibility: SettingsVisibility): SettingsSection[] =>
  SETTINGS_SECTIONS.filter((section) => {
    if (INSTANCE_SECTIONS.includes(section)) return visibility.isInstanceAdmin;
    if (section === "roles") return visibility.showRoles;
    if (section === "plays") return visibility.showPlays;
    if (section === "interview") return visibility.showInterviewTemplate;
    if (section === "mentions") return visibility.showMentionLayout;
    return true;
  });

interface SettingsNavProps {
  active: SettingsSection;
  sections: SettingsSection[];
}

export const SettingsNav = ({ active, sections }: SettingsNavProps) => {
  const items: SettingsSectionNavItem[] = sections.map((section) => ({
    section,
    label: sectionLabels[section],
    danger: section === "danger",
    group: INSTANCE_SECTIONS.includes(section) ? INSTANCE_GROUP : WORKSPACE_GROUP,
  }));

  return <SettingsSectionNav ariaLabel="Configuration sections" basePath="/configuration" active={active} items={items} />;
};
