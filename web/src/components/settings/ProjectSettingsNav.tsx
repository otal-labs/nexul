import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { projectSettingsPath, projectToken, type Project } from "@/models/Project";

export const PROJECT_SETTINGS_SECTIONS = [
  "general",
  "access",
  "statuses",
  "categories",
  "types",
  "labels",
  "repositories",
  "services",
  "danger",
] as const;

export type ProjectSettingsSection = (typeof PROJECT_SETTINGS_SECTIONS)[number];

export const DEFAULT_PROJECT_SETTINGS_SECTION: ProjectSettingsSection = "general";

export const isProjectSettingsSection = (value: string | null | undefined): value is ProjectSettingsSection =>
  !!value && (PROJECT_SETTINGS_SECTIONS as readonly string[]).includes(value);

export const sectionLabels: Record<ProjectSettingsSection, string> = {
  general: "General",
  access: "Access",
  statuses: "Statuses",
  categories: "Categories",
  types: "Ticket types",
  labels: "Labels",
  repositories: "Repositories",
  services: "Services",
  danger: "Danger zone",
};

const sectionGroups: Record<ProjectSettingsSection, string | undefined> = {
  general: "Project",
  access: "Project",
  statuses: "Board",
  categories: "Board",
  types: "Board",
  labels: "Board",
  repositories: "Delivery",
  services: "Delivery",
  danger: undefined,
};

// Access is who may open the project, so only someone who can change that sees it; for anyone else it is not a section.
export const visibleProjectSettingsSections = (canManageAccess: boolean): ProjectSettingsSection[] =>
  PROJECT_SETTINGS_SECTIONS.filter((section) => section !== "access" || canManageAccess);

interface ProjectSettingsNavProps {
  project: Project;
  active: ProjectSettingsSection;
  canManageAccess: boolean;
}

export const ProjectSettingsNav = ({ project, active, canManageAccess }: ProjectSettingsNavProps) => {
  const wsPath = useWorkspacePath();
  return (
    <SettingsSectionNav
      ariaLabel="Project settings sections"
      basePath={wsPath(projectSettingsPath(projectToken(project)))}
      active={active}
      items={visibleProjectSettingsSections(canManageAccess).map((section) => ({
        section,
        label: sectionLabels[section],
        group: sectionGroups[section],
        danger: section === "danger",
      }))}
    />
  );
};
