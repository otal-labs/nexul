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
  // Team also opens to anyone who manages members in a workspace, scoped to those workspaces.
  showTeam: boolean;
}

// The sections the viewer may open, in nav order; the page falls back to the first when the URL names none of them.
export const visibleSettingsSections = (visibility: SettingsVisibility): SettingsSection[] => {
  const sections = SETTINGS_SECTIONS.filter((section) => {
    if (section === "team") return visibility.isInstanceAdmin || visibility.showTeam;
    if (INSTANCE_SECTIONS.includes(section)) return visibility.isInstanceAdmin;
    if (section === "roles") return visibility.showRoles;
    if (section === "plays") return visibility.showPlays;
    if (section === "interview") return visibility.showInterviewTemplate;
    if (section === "mentions") return visibility.showMentionLayout;
    return true;
  });
  if (visibility.isInstanceAdmin || !sections.includes("team")) return sections;
  // Without the instance, Team is a workspace-scoped section, so it sits with them ahead of Danger zone.
  const workspaceOnly: SettingsSection[] = sections.filter((section) => section !== "team");
  workspaceOnly.splice(workspaceOnly.indexOf("danger"), 0, "team");
  return workspaceOnly;
};

interface SettingsNavProps {
  active: SettingsSection;
  sections: SettingsSection[];
  isInstanceAdmin: boolean;
}

const groupOf = (section: SettingsSection, isInstanceAdmin: boolean): string => {
  if (section === "team") return isInstanceAdmin ? INSTANCE_GROUP : WORKSPACE_GROUP;
  return INSTANCE_SECTIONS.includes(section) ? INSTANCE_GROUP : WORKSPACE_GROUP;
};

export const SettingsNav = ({ active, sections, isInstanceAdmin }: SettingsNavProps) => {
  const items: SettingsSectionNavItem[] = sections.map((section) => ({
    section,
    label: sectionLabels[section],
    danger: section === "danger",
    group: groupOf(section, isInstanceAdmin),
  }));

  return <SettingsSectionNav ariaLabel="Configuration sections" basePath="/configuration" active={active} items={items} />;
};
