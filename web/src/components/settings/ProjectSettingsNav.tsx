import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";

export const PROJECT_SETTINGS_SECTIONS = [
  "general",
  "categories",
  "repositories",
  "services",
  "board",
  "pairing",
  "danger",
] as const;

export type ProjectSettingsSection = (typeof PROJECT_SETTINGS_SECTIONS)[number];

export const DEFAULT_PROJECT_SETTINGS_SECTION: ProjectSettingsSection = "general";

export const isProjectSettingsSection = (value: string | null | undefined): value is ProjectSettingsSection =>
  !!value && (PROJECT_SETTINGS_SECTIONS as readonly string[]).includes(value);

const sectionLabels: Record<ProjectSettingsSection, string> = {
  general: "General",
  categories: "Categories",
  repositories: "Repositories",
  services: "Services",
  board: "Board",
  pairing: "T3 pairing",
  danger: "Danger zone",
};

interface ProjectSettingsNavProps {
  active: ProjectSettingsSection;
}

export const ProjectSettingsNav = ({ active }: ProjectSettingsNavProps) => (
  <SettingsSectionNav
    ariaLabel="Project settings sections"
    active={active}
    items={PROJECT_SETTINGS_SECTIONS.map((section) => ({
      section,
      label: sectionLabels[section],
      danger: section === "danger",
    }))}
  />
);
