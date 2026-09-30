import { SettingsSectionNav, type SettingsSectionNavItem } from "@/components/settings/SettingsSectionNav";
import type { InstanceSection } from "@/models/Access";

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

export const INSTANCE_GROUP = "Instance settings";

const INSTANCE_SECTIONS: readonly SettingsSection[] = ["instance", "team", "sign-in", "connectors", "dns"];

export const sectionLabels: Record<SettingsSection, string> = {
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
  // The instance sections whose permission the viewer holds in some workspace (models/Access.tsx).
  instanceSections: readonly InstanceSection[];
  // Team opens to an accounts:read holder, who sees everyone in Settings, and to anyone who manages
  // members in a workspace, who sees those workspaces in Configuration.
  showTeam: boolean;
  teamIsInstanceWide: boolean;
  // Each gated workspace section renders only for its permission holder; the nav must not link to an empty section.
  showRoles: boolean;
  showPlays: boolean;
  showInterviewTemplate: boolean;
  showMentionLayout: boolean;
}

// The sections the viewer may open, in nav order; the page falls back to the first when the URL names none of them.
export const visibleSettingsSections = (visibility: SettingsVisibility): SettingsSection[] => {
  const sections = SETTINGS_SECTIONS.filter((section) => {
    if (section === "team") return visibility.showTeam;
    if (INSTANCE_SECTIONS.includes(section)) return (visibility.instanceSections as readonly string[]).includes(section);
    if (section === "roles") return visibility.showRoles;
    if (section === "plays") return visibility.showPlays;
    if (section === "interview") return visibility.showInterviewTemplate;
    if (section === "mentions") return visibility.showMentionLayout;
    return true;
  });
  if (visibility.teamIsInstanceWide || !sections.includes("team")) return sections;
  // Without accounts:read, Team is a workspace-scoped section, so it sits with them ahead of Danger zone.
  const workspaceOnly: SettingsSection[] = sections.filter((section) => section !== "team");
  workspaceOnly.splice(workspaceOnly.indexOf("danger"), 0, "team");
  return workspaceOnly;
};

// Instance sections live on the Settings page; Team is one of them only for an accounts:read holder.
export const isInstanceSection = (section: SettingsSection, teamIsInstanceWide: boolean): boolean => {
  if (section === "team") return teamIsInstanceWide;
  return INSTANCE_SECTIONS.includes(section);
};

interface SettingsNavProps {
  active: SettingsSection;
  sections: SettingsSection[];
}

export const SettingsNav = ({ active, sections }: SettingsNavProps) => {
  const items: SettingsSectionNavItem[] = sections.map((section) => ({
    section,
    label: sectionLabels[section],
    danger: section === "danger",
  }));

  return <SettingsSectionNav ariaLabel="Configuration sections" basePath="/configuration" active={active} items={items} />;
};
